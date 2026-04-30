package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"sonoscli-gui/internal/sonos"
)

//go:embed static/*
var staticFiles embed.FS

type WebServer struct {
	Topology sonos.Topology
	Clients  map[string]*sonos.Client
	
	upgrader websocket.Upgrader
	hub      *Hub
	server   *http.Server
	cancel   context.CancelFunc

	OnTopologyUpdate func(sonos.Topology)
	
	callbackURL string
	mu          sync.RWMutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewWebServer() *WebServer {
	return &WebServer{
		Clients: make(map[string]*sonos.Client),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		hub: newHub(),
	}
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 100),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					slog.Warn("Client buffer full, dropping message")
				}
			}
		}
	}
}

func (s *WebServer) Start(port int) error {
	s.mu.Lock()
	if s.server != nil {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	go s.hub.Run(ctx)
	go s.startEventListener(ctx)

	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// API
	mux.HandleFunc("/api/topology", s.handleTopology)
	mux.HandleFunc("/api/room/", s.handleRoom)
	mux.HandleFunc("/ws", s.handleWS)

	addr := fmt.Sprintf(":%d", port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	slog.Info("Web server starting", "addr", addr)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *WebServer) Stop() error {
	s.mu.Lock()
	if s.server == nil {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	
	ctx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	
	err := s.server.Shutdown(ctx)
	s.mu.Lock()
	s.server = nil
	s.cancel = nil
	s.mu.Unlock()
	slog.Info("Web server stopped")
	return err
}

func (s *WebServer) startEventListener(ctx context.Context) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		slog.Error("Failed to start event listener", "err", err)
		return
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port
	ip := getLocalIP()
	callbackURL := fmt.Sprintf("http://%s:%d/notify", ip, port)
	slog.Info("Event listener started", "url", callbackURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", s.handleNotify)
	server := &http.Server{Handler: mux}

	// Store callback URL for subscription
	s.mu.Lock()
	s.callbackURL = callbackURL
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	server.Serve(l)
}

func (s *WebServer) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "NOTIFY" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.BroadcastRefreshHint()
	w.WriteHeader(http.StatusOK)
}

func (s *WebServer) BroadcastRefreshHint() {
	select {
	case s.hub.broadcast <- []byte(`{"type":"refresh_hint"}`):
	default:
	}
}

func (s *WebServer) BroadcastRoomUpdate(roomID string, status RoomStatus) {
	slog.Debug("Broadcasting room update", "room", roomID, "track", status.Track)
	data, _ := json.Marshal(map[string]interface{}{
		"type":    "room_update",
		"roomId":  roomID,
		"payload": status,
	})
	select {
	case s.hub.broadcast <- data:
	default:
		slog.Warn("Broadcast hub busy, dropping update", "room", roomID)
	}
}

func (s *WebServer) UpdateTopology(top sonos.Topology) {
	s.mu.Lock()
	s.Topology = top
	
	// Subscribe to all coordinators for events if we have a callback URL
	cb := s.callbackURL
	if cb != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, g := range top.Groups {
			ip := g.Coordinator.IP
			go func(targetIP string) {
				client := sonos.NewClient(targetIP, 2*time.Second)
				client.SubscribeAVTransport(ctx, cb, 300*time.Second)
				client.SubscribeRenderingControl(ctx, cb, 300*time.Second)
			}(ip)
		}
	}

	if s.OnTopologyUpdate != nil {
		s.OnTopologyUpdate(top)
	}
	s.mu.Unlock()
	s.broadcastTopology()
}

func (s *WebServer) broadcastTopology() {
	s.mu.RLock()
	data, _ := json.Marshal(map[string]interface{}{
		"type":    "topology",
		"payload": s.Topology,
	})
	s.mu.RUnlock()
	select {
	case s.hub.broadcast <- data:
	default:
	}
}

func (s *WebServer) handleTopology(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Topology)
}

func (s *WebServer) handleRoom(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 4 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	roomID := parts[3]
	
	s.mu.RLock()
	var targetIP string
	var roomName string
	for _, g := range s.Topology.Groups {
		if g.ID == roomID {
			targetIP = g.Coordinator.IP
			roomName = g.Coordinator.Name
			break
		}
		for _, m := range g.Members {
			if m.UUID == roomID {
				targetIP = m.IP
				roomName = m.Name
				break
			}
		}
	}
	s.mu.RUnlock()

	if targetIP == "" {
		slog.Warn("Web request for unknown room", "roomID", roomID)
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	client := sonos.NewClient(targetIP, 2*time.Second)

	if len(parts) >= 5 && parts[4] == "status" {
		s.handleRoomStatus(w, r, client, roomID, roomName)
		return
	}

	if len(parts) >= 5 && parts[4] == "control" {
		s.handleRoomControl(w, r, client, roomID)
		return
	}

	if len(parts) >= 5 && parts[4] == "queue" {
		s.handleRoomQueue(w, r, client)
		return
	}

	if len(parts) >= 5 && parts[4] == "favorites" {
		s.handleRoomFavorites(w, r, client)
		return
	}

	if len(parts) >= 5 && parts[4] == "inputs" {
		s.handleRoomInputs(w, r, client, roomID)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func (s *WebServer) handleRoomQueue(w http.ResponseWriter, r *http.Request, client *sonos.Client) {
	res, err := client.Browse(r.Context(), "Q:0", 0, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items, _ := sonos.ParseDIDLItems(res.Result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *WebServer) handleRoomFavorites(w http.ResponseWriter, r *http.Request, client *sonos.Client) {
	res, err := client.Browse(r.Context(), "FV:2", 0, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items, _ := sonos.ParseDIDLItems(res.Result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

type RoomStatus struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Track              string `json:"track"`
	Artist             string `json:"artist"`
	Album              string `json:"album"`
	AlbumArt           string `json:"albumArt"`
	State              string `json:"state"`
	Volume             int    `json:"volume"`
	PositionSeconds    int    `json:"positionSeconds"`
	DurationSeconds    int    `json:"durationSeconds"`
	Bass               int    `json:"bass"`
	Treble             int    `json:"treble"`
	Loudness           bool   `json:"loudness"`
	NightMode          bool   `json:"nightMode"`
	SpeechEnhancement bool   `json:"speechEnhancement"`
	GroupVolume        int    `json:"groupVolume"`
	GroupMute          bool   `json:"groupMute"`
}

func (s *WebServer) handleRoomStatus(w http.ResponseWriter, r *http.Request, client *sonos.Client, id, name string) {
	status := s.BuildRoomStatus(r.Context(), client, id, name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *WebServer) BuildRoomStatus(ctx context.Context, client *sonos.Client, id, name string) RoomStatus {
	pos, _ := client.GetPositionInfo(ctx)
	
	title := "No Track"
	artist := ""
	album := ""
	art := ""

	uri := strings.ToLower(pos.TrackURI)
	if strings.Contains(uri, "htastream") || strings.Contains(uri, "hdmi") || strings.Contains(uri, "spdif") || strings.Contains(uri, "arc") {
		title = "TV Audio"
	} else if strings.Contains(uri, "line-in") || strings.Contains(uri, "linein") || strings.Contains(uri, "analog") {
		title = "Line-In"
	} else if pos.TrackMeta != "" {
		if it, ok := sonos.ParseNowPlaying(pos.TrackMeta); ok {
			title = it.Title
			artist = it.Artist
			album = it.Album
			if it.AlbumArtURI != "" {
				art = sonos.AlbumArtURL(client.IP, it.AlbumArtURI)
			}
		}
	}

	status := RoomStatus{
		ID:                id,
		Name:              name,
		Track:             title,
		Artist:            artist,
		Album:             album,
		AlbumArt:          art,
		PositionSeconds:   parseDuration(pos.RelTime),
		DurationSeconds:   parseDuration(pos.TrackDuration),
	}

	transport, _ := client.GetTransportInfo(ctx)
	status.State = transport.State

	// Rendering
	vol, _ := client.GetVolume(ctx)
	status.Volume = vol

	bass, _ := client.GetEQ(ctx, "Bass")
	status.Bass = bass
	treble, _ := client.GetEQ(ctx, "Treble")
	status.Treble = treble
	loud, _ := client.GetLoudness(ctx)
	status.Loudness = loud

	// Device properties
	night, _ := client.GetNightMode(ctx)
	status.NightMode = night
	speech, _ := client.GetSpeechEnhancement(ctx)
	status.SpeechEnhancement = speech

	gvol, _ := client.GetGroupVolume(ctx)
	status.GroupVolume = gvol
	gmute, _ := client.GetGroupMute(ctx)
	status.GroupMute = gmute

	return status
}


func (s *WebServer) handleRoomControl(w http.ResponseWriter, r *http.Request, client *sonos.Client, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string `json:"command"`
		Value   int    `json:"value"`
		URI     string `json:"uri"`
		Meta    string `json:"meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("Web control request", "command", req.Command, "room", id, "value", req.Value, "uri", req.URI)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var err error
	switch req.Command {
	case "play": err = client.Play(ctx)
	case "pause": err = client.Pause(ctx)
	case "playpause":
		info, _ := client.GetTransportInfo(ctx)
		if info.State == "PLAYING" {
			err = client.Pause(ctx)
		} else {
			err = client.Play(ctx)
		}
	case "next": err = client.Next(ctx)
	case "previous": err = client.Previous(ctx)
	case "volume": err = client.SetVolume(ctx, req.Value)
	case "bass": err = client.SetEQ(ctx, "Bass", req.Value)
	case "treble": err = client.SetEQ(ctx, "Treble", req.Value)
	case "loudness":
		l, _ := client.GetLoudness(ctx)
		err = client.SetLoudness(ctx, !l)
	case "nightmode":
		n, _ := client.GetNightMode(ctx)
		err = client.SetNightMode(ctx, !n)
	case "speechenhancement":
		se, _ := client.GetSpeechEnhancement(ctx)
		err = client.SetSpeechEnhancement(ctx, !se)
	case "seek":
		err = client.SeekRelTime(ctx, formatDuration(req.Value))
	case "playtrack":
		trackStr := req.URI
		if trackStr == "" {
			trackStr = fmt.Sprintf("%d", req.Value)
		}
		if idx := strings.LastIndex(trackStr, "/"); idx != -1 {
			trackStr = trackStr[idx+1:]
		}
		var trackNum int
		fmt.Sscanf(trackStr, "%d", &trackNum)
		if trackNum > 0 {
			err = client.SeekTrackNumber(ctx, trackNum)
			if err == nil {
				err = client.Play(ctx)
			}
		}
	case "playuri":
		err = client.PlayURI(ctx, req.URI, req.Meta)
	case "joingroup":
		err = client.JoinGroup(ctx, req.Meta)
	case "leavegroup":
		err = client.LeaveGroup(ctx)
	case "groupvolume":
		err = client.SetGroupVolume(ctx, req.Value)
	case "groupmute":
		m, _ := client.GetGroupMute(ctx)
		err = client.SetGroupMute(ctx, !m)
	}

	if err != nil {
		slog.Error("Web control failed", "command", req.Command, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger immediate broadcast of new state
	s.mu.RLock()
	roomName := ""
	for _, g := range s.Topology.Groups {
		if g.ID == id {
			roomName = g.Coordinator.Name
			break
		}
	}
	s.mu.RUnlock()
	
	// Tiny delay to let hardware state settle before fetching new status
	time.Sleep(150 * time.Millisecond)
	
	status := s.BuildRoomStatus(ctx, client, id, roomName)
	
	// Optimistic update for modes that might fail to report their state via GetEQ
	switch req.Command {
	case "loudness":
		status.Loudness = !status.Loudness
	case "nightmode":
		status.NightMode = !status.NightMode
	case "speechenhancement":
		status.SpeechEnhancement = !status.SpeechEnhancement
	}
	
	s.BroadcastRoomUpdate(id, status)

	w.WriteHeader(http.StatusOK)
}

func parseDuration(s string) int {
	parts := (func() []string {
		var res []string
		curr := ""
		for i := 0; i < len(s); i++ {
			if s[i] == ':' {
				res = append(res, curr)
				curr = ""
			} else {
				curr += string(s[i])
			}
		}
		res = append(res, curr)
		return res
	}())
	if len(parts) != 3 {
		return 0
	}
	var h, m, sec int
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)
	fmt.Sscanf(parts[2], "%d", &sec)
	return h*3600 + m*60 + sec
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (s *WebServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WS upgrade failed", "err", err)
		return
	}
	client := &Client{hub: s.hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func (s *WebServer) handleRoomInputs(w http.ResponseWriter, r *http.Request, client *sonos.Client, id string) {
	slog.Info("Web request for inputs", "roomID", id, "ip", client.IP)
	ctx := r.Context()
	
	type Input struct {
		Title string `json:"title"`
		URI   string `json:"uri"`
		Meta  string `json:"meta"`
	}
	var inputs []Input

	// Always add TV Input as a possibility if it's a RINCON ID
	if strings.HasPrefix(id, "RINCON_") {
		uuid := id
		if idx := strings.Index(uuid, ":"); idx != -1 {
			uuid = uuid[:idx]
		}
		inputs = append(inputs, Input{
			Title: "TV Input",
			URI:   "x-sonos-htastream:" + uuid + ":spdif",
		})
	}

	res, err := client.Browse(ctx, "AI:", 0, 50)
	if err == nil {
		items, _ := sonos.ParseDIDLItems(res.Result)
		slog.Info("Found inputs", "count", len(items), "roomID", id)
		for _, it := range items {
			inputs = append(inputs, Input{
				Title: it.Title,
				URI:   it.URI,
				Meta:  it.ResMD,
			})
		}
	} else {
		slog.Warn("Failed to browse inputs", "err", err, "roomID", id)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inputs)
}

func splitPath(path string) []string {
	var parts []string
	curr := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			parts = append(parts, curr)
			curr = ""
		} else {
			curr += string(path[i])
		}
	}
	parts = append(parts, curr)
	
	var res []string
	for _, p := range parts {
		if p != "" {
			res = append(res, p)
		}
	}
	return append([]string{""}, res...)
}
