package ui

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"sonoscli-gui/internal/sonos"
)

type EventManager struct {
	sa           *SonosApp
	listener     net.Listener
	server       *http.Server
	callbackURL  string
	
	subsMu       sync.Mutex
	subs         map[string]sonos.Subscription
}

func NewEventManager(sa *SonosApp) *EventManager {
	return &EventManager{
		sa:   sa,
		subs: make(map[string]sonos.Subscription),
	}
}

func (em *EventManager) Start() error {
	l, err := net.Listen("tcp", ":0") // Listen on random port
	if err != nil {
		return err
	}
	em.listener = l
	
	port := l.Addr().(*net.TCPAddr).Port
	ip := getLocalIP()
	em.callbackURL = "http://" + ip + ":" + strings.TrimPrefix(l.Addr().String(), "[::]")
	// Actually, l.Addr().String() might be ":PORT", so we need local IP
	em.callbackURL = "http://" + ip + ":" + strconv.Itoa(port) + "/notify"

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", em.handleNotify)
	
	em.server = &http.Server{Handler: mux}
	
	go em.server.Serve(l)
	slog.Info("Event listener started", "url", em.callbackURL)
	return nil
}

func (em *EventManager) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "NOTIFY" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	sid := r.Header.Get("SID")
	_, _ = io.ReadAll(r.Body) // Consume body to avoid leak
	
	slog.Debug("Received event notification", "sid", sid)
	
	// We don't even need to map SID if we just update the current room, 
	// but better to be precise. For now, let's just trigger a room refresh.
	if em.sa.CurrentRoomID != "" {
		ui := em.sa.RoomUI[em.sa.CurrentRoomID]
		if ui != nil {
			// Find IP for current room
			var ip string
			for _, g := range em.sa.Topology.Groups {
				if g.ID == em.sa.CurrentRoomID {
					ip = g.Coordinator.IP
					break
				}
				for _, m := range g.Members {
					if m.UUID == em.sa.CurrentRoomID {
						ip = m.IP
						break
					}
				}
			}
			if ip != "" {
				go em.sa.updateRoomStatus(ui, ip)
			}
		}
	}
	
	w.WriteHeader(http.StatusOK)
}

func (em *EventManager) Subscribe(ctx context.Context, ip string) {
	client := sonos.NewClient(ip, 2*time.Second)
	
	sub, err := client.SubscribeAVTransport(ctx, em.callbackURL, 300*time.Second)
	if err == nil {
		em.subsMu.Lock()
		em.subs[sub.SID] = sub
		em.subsMu.Unlock()
		slog.Info("Subscribed to AVTransport", "ip", ip, "sid", sub.SID)
	} else {
		slog.Error("Failed to subscribe", "ip", ip, "err", err)
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
