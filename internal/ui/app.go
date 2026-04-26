package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"sonoscli-gui/internal/appconfig"
	"sonoscli-gui/internal/scenes"
	"sonoscli-gui/internal/sonos"
)

type SonosApp struct {
	App    fyne.App
	Window fyne.Window

	// Discovery & Topology
	Topology sonos.Topology
	Client   *sonos.Client

	// UI Components
	Sidebar     *widget.Tree
	MainArea    *fyne.Container
	StatusLabel *widget.Label

	// Now Playing State
	CurrentRoomID string
	RoomUI        map[string]*RoomUI
	
	// Scenes
	ScenesStore scenes.Store
	ScenesList  *fyne.Container

	// SMAPI
	TokenStore sonos.SMAPITokenStore

	// Config
	ConfigStore appconfig.Store
	Config      appconfig.Config

	// Events
	Events *EventManager
}

type RoomUI struct {
	Container    *fyne.Container
	NameLabel    *widget.Label
	TrackLabel   *widget.Label
	ArtistLabel  *widget.Label
	AlbumArt     *canvas.Image
	PlayPauseBtn *widget.Button
	VolSlider    *widget.Slider
	
	// Progress
	Progress     *widget.ProgressBar
	TimeLabel    *widget.Label

	// Tabs
	Tabs *container.AppTabs
	
	// Lists & Containers
	QueueList     *fyne.Container
	FavoritesList *fyne.Container
	GroupList     *fyne.Container
	AudioSettings *fyne.Container
	InputsList    *fyne.Container
	
	// Audio Settings Widgets
	BassSlider   *widget.Slider
	TrebleSlider *widget.Slider
	LoudnessBtn  *widget.Button
	NightBtn     *widget.Button
	SpeechBtn    *widget.Button
	
	// Line Out (Port/Connect)
	LineOutSelect *widget.Select
	LineOutCard   fyne.CanvasObject
	
	// Group Volume
	GroupVolSlider *widget.Slider
	GroupMuteBtn   *widget.Button
}

func NewSonosApp(a fyne.App, w fyne.Window) *SonosApp {
	ss, _ := scenes.NewFileStore()
	ts, _ := sonos.NewDefaultSMAPITokenStore()
	cs, _ := appconfig.NewDefaultStore()
	cfg, _ := cs.Load()

	sa := &SonosApp{
		App:         a,
		Window:      w,
		RoomUI:      make(map[string]*RoomUI),
		ScenesStore: ss,
		TokenStore:  ts,
		ConfigStore: cs,
		Config:      cfg,
	}
	sa.Events = NewEventManager(sa)
	sa.Events.Start()

	sa.buildUI()
	return sa
}

func makeCard(title string, content fyne.CanvasObject) fyne.CanvasObject {
	cardBg := canvas.NewRectangle(theme.InputBackgroundColor())
	cardBg.CornerRadius = 10
	
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewPadded(titleLabel)
	
	return container.NewStack(
		cardBg,
		container.NewBorder(header, nil, nil, nil, container.NewPadded(content)),
	)
}

func (sa *SonosApp) buildUI() {
	sa.StatusLabel = widget.NewLabel("Initializing...")

	// Sidebar (Tree for Groups/Rooms)
	sa.Sidebar = widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			if id == "" {
				return []widget.TreeNodeID{"root"}
			}
			if id == "root" {
				ids := make([]widget.TreeNodeID, len(sa.Topology.Groups))
				for i, g := range sa.Topology.Groups {
					ids[i] = g.ID
				}
				return ids
			}
			for _, g := range sa.Topology.Groups {
				if g.ID == id {
					ids := make([]widget.TreeNodeID, len(g.Members))
					for i, m := range g.Members {
						ids[i] = m.UUID
					}
					return ids
				}
			}
			return nil
		},
		func(id widget.TreeNodeID) bool {
			if id == "" || id == "root" {
				return true
			}
			for _, g := range sa.Topology.Groups {
				if g.ID == id {
					return true
				}
			}
			return false
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id == "root" {
				label.SetText("Sonos System")
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			// Check if it's a group
			for _, g := range sa.Topology.Groups {
				if g.ID == id {
					label.SetText(fmt.Sprintf("Group: %s", g.Coordinator.Name))
					label.TextStyle = fyne.TextStyle{Bold: true}
					return
				}
			}
			// Must be a room
			for _, g := range sa.Topology.Groups {
				for _, m := range g.Members {
					if m.UUID == id {
						label.SetText(m.Name)
						if m.IsCoordinator {
							label.TextStyle = fyne.TextStyle{Italic: true}
						} else {
							label.TextStyle = fyne.TextStyle{}
						}
						return
					}
				}
			}
			label.SetText(id)
		},
	)

	sa.Sidebar.OnSelected = func(id widget.TreeNodeID) {
		if id == "" || id == "root" { return }
		sa.showRoom(id)
	}

	sa.MainArea = container.NewStack(widget.NewLabel("Select a room to start"))

	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		go sa.Discover()
	})
	
	manualIP := widget.NewEntry()
	manualIP.SetPlaceHolder("Manual IP")
	manualBtn := widget.NewButton("Connect", func() {
		if manualIP.Text != "" {
			go sa.DiscoverIP(manualIP.Text)
		}
	})
	
	manualContainer := container.NewBorder(nil, nil, nil, manualBtn, manualIP)

	sa.ScenesList = container.NewVBox()
	scenesScroll := container.NewVScroll(sa.ScenesList)
	
	saveSceneBtn := widget.NewButtonWithIcon("Save Current as Scene", theme.DocumentSaveIcon(), sa.saveCurrentScene)
	
	scenesTab := container.NewBorder(saveSceneBtn, nil, nil, nil, scenesScroll)

	left := container.NewBorder(container.NewVBox(container.NewHBox(widget.NewLabel("Rooms"), layout.NewSpacer(), refreshBtn), manualContainer), nil, nil, nil, sa.Sidebar)
	
	sidebarTabs := container.NewAppTabs(
		container.NewTabItem("Rooms", left),
		container.NewTabItem("Scenes", scenesTab),
	)

	split := container.NewHSplit(sidebarTabs, sa.MainArea)
	split.Offset = 0.3

	sa.Window.SetContent(container.NewBorder(nil, sa.StatusLabel, nil, nil, split))
	
	go sa.refreshScenes()
}

func (sa *SonosApp) Start() {
	// 1. Try known IPs first
	if len(sa.Config.KnownIPs) > 0 {
		for _, ip := range sa.Config.KnownIPs {
			go sa.DiscoverIP(ip)
		}
	}
	// 2. Run background discovery
	go sa.Discover()
}

func (sa *SonosApp) updateConfig(ip string, roomID string) {
	changed := false
	if ip != "" {
		found := false
		for _, k := range sa.Config.KnownIPs {
			if k == ip { found = true; break }
		}
		if !found {
			sa.Config.KnownIPs = append(sa.Config.KnownIPs, ip)
			changed = true
		}
	}
	if roomID != "" && sa.Config.LastRoomID != roomID {
		sa.Config.LastRoomID = roomID
		changed = true
	}

	if changed {
		sa.ConfigStore.Save(sa.Config)
	}
}

func (sa *SonosApp) DiscoverIP(ip string) {
	fyne.Do(func() {
		sa.StatusLabel.SetText(fmt.Sprintf("Connecting to %s...", ip))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := sonos.NewClient(ip, 2*time.Second)
	sa.Client = client
	
	top, err := client.GetTopology(ctx)
	if err != nil {
		fyne.Do(func() {
			sa.StatusLabel.SetText(fmt.Sprintf("Connection error: %v", err))
		})
		return
	}

	sa.Topology = top
	sa.updateConfig(ip, "")
	
	fyne.Do(func() {
		sa.StatusLabel.SetText(fmt.Sprintf("Found %d groups and %d rooms via manual IP.", len(top.Groups), len(top.ByName)))
		sa.Sidebar.Refresh()
		sa.Sidebar.OpenAllBranches()
		
		if sa.CurrentRoomID == "" && sa.Config.LastRoomID != "" {
			sa.showRoom(sa.Config.LastRoomID)
		}
	})
}

func (sa *SonosApp) Discover() {
	fyne.Do(func() {
		sa.StatusLabel.SetText("Discovering speakers...")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	devices, err := sonos.Discover(ctx, sonos.DiscoverOptions{Timeout: 10 * time.Second})
	if err != nil {
		fyne.Do(func() {
			sa.StatusLabel.SetText(fmt.Sprintf("Discovery error: %v", err))
		})
		return
	}

	if len(devices) == 0 {
		fyne.Do(func() {
			sa.StatusLabel.SetText("No speakers found.")
		})
		return
	}

	client := sonos.NewClient(devices[0].IP, 2*time.Second)
	sa.Client = client
	
	top, err := client.GetTopology(ctx)
	if err != nil {
		fyne.Do(func() {
			sa.StatusLabel.SetText(fmt.Sprintf("Topology error: %v", err))
		})
		return
	}

	sa.Topology = top
	for _, dev := range devices {
		sa.updateConfig(dev.IP, "")
	}

	fyne.Do(func() {
		sa.StatusLabel.SetText(fmt.Sprintf("Found %d groups and %d rooms.", len(top.Groups), len(top.ByName)))
		sa.Sidebar.Refresh()
		sa.Sidebar.OpenAllBranches()

		if sa.CurrentRoomID == "" && sa.Config.LastRoomID != "" {
			sa.showRoom(sa.Config.LastRoomID)
		}
	})
}

func (sa *SonosApp) showRoom(id string) {
	sa.CurrentRoomID = id
	sa.updateConfig("", id)
	
	var targetIP string
	var name string

	for _, g := range sa.Topology.Groups {
		if g.ID == id {
			targetIP = g.Coordinator.IP
			name = g.Coordinator.Name
			break
		}
		for _, m := range g.Members {
			if m.UUID == id {
				targetIP = m.IP
				name = m.Name
				break
			}
		}
	}

	if targetIP == "" {
		return
	}

	ui, ok := sa.RoomUI[id]
	if !ok {
		ui = sa.createRoomUI(name, targetIP, id)
		sa.RoomUI[id] = ui
	}

	fyne.Do(func() {
		sa.MainArea.Objects = []fyne.CanvasObject{ui.Container}
		sa.MainArea.Refresh()
	})
	
	go sa.updateRoomStatus(ui, targetIP)
	go sa.updateAudioSettingsUI(ui, targetIP)
	go sa.updateFavorites(ui, targetIP)
	go sa.updateQueue(ui, targetIP)
	go sa.updateInputsUI(ui, targetIP)
	go sa.updateGroups(ui, targetIP, id)
}

func (sa *SonosApp) createRoomUI(name, ip, id string) *RoomUI {
	ui := &RoomUI{}
	
	ui.NameLabel = widget.NewLabelWithStyle(name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	ui.TrackLabel = widget.NewLabelWithStyle("No Track", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	ui.ArtistLabel = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	ui.AlbumArt = canvas.NewImageFromResource(theme.BrokenImageIcon())
	ui.AlbumArt.FillMode = canvas.ImageFillContain
	ui.AlbumArt.SetMinSize(fyne.NewSize(250, 250))

	ui.Progress = widget.NewProgressBar()
	ui.Progress.Max = 100
	ui.TimeLabel = widget.NewLabelWithStyle("0:00 / 0:00", fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})

	client := sonos.NewClient(ip, 2*time.Second)

	ui.PlayPauseBtn = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		go func() {
			ctx := context.Background()
			info, _ := client.GetTransportInfo(ctx)
			if info.State == "PLAYING" {
				client.Pause(ctx)
			} else {
				client.Play(ctx)
			}
			sa.updateRoomStatus(ui, ip)
		}()
	})
	ui.PlayPauseBtn.Importance = widget.HighImportance

	ui.VolSlider = widget.NewSlider(0, 100)
	ui.VolSlider.OnChanged = func(v float64) {
		go client.SetVolume(context.Background(), int(v))
	}

	playbackControls := container.NewCenter(container.NewHBox(
		widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
			client.Previous(context.Background())
			sa.updateRoomStatus(ui, ip)
		}),
		ui.PlayPauseBtn,
		widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
			client.Next(context.Background())
			sa.updateRoomStatus(ui, ip)
		}),
	))

	nowPlayingInfo := container.NewVBox(
		ui.TrackLabel,
		ui.ArtistLabel,
		container.NewPadded(container.NewVBox(ui.Progress, ui.TimeLabel)),
		playbackControls,
		container.NewPadded(container.NewBorder(nil, nil, widget.NewIcon(theme.VolumeDownIcon()), widget.NewIcon(theme.VolumeUpIcon()), ui.VolSlider)),
	)

	nowPlaying := container.NewAdaptiveGrid(2,
		container.NewPadded(container.NewCenter(ui.AlbumArt)),
		container.NewCenter(nowPlayingInfo),
	)

	// Audio Settings
	ui.BassSlider = widget.NewSlider(-10, 10)
	ui.BassSlider.OnChanged = func(v float64) { go client.SetEQ(context.Background(), "Bass", int(v)) }
	ui.TrebleSlider = widget.NewSlider(-10, 10)
	ui.TrebleSlider.OnChanged = func(v float64) { go client.SetEQ(context.Background(), "Treble", int(v)) }
	
	ui.LoudnessBtn = widget.NewButton("Loudness", func() {
		go func() {
			ctx := context.Background()
			l, _ := client.GetLoudness(ctx)
			client.SetLoudness(ctx, !l)
			sa.updateAudioSettingsUI(ui, ip)
		}()
	})
	ui.NightBtn = widget.NewButton("Night Mode", func() {
		go func() {
			ctx := context.Background()
			n, _ := client.GetNightMode(ctx)
			client.SetNightMode(ctx, !n)
			sa.updateAudioSettingsUI(ui, ip)
		}()
	})
	ui.SpeechBtn = widget.NewButton("Speech Enhancement", func() {
		go func() {
			ctx := context.Background()
			s, _ := client.GetSpeechEnhancement(ctx)
			client.SetSpeechEnhancement(ctx, !s)
			sa.updateAudioSettingsUI(ui, ip)
		}()
	})

	ui.LineOutSelect = widget.NewSelect([]string{"Variable", "Fixed", "Pass-Through"}, func(s string) {
		mode := 0
		switch s {
		case "Variable": mode = 0
		case "Fixed": mode = 1
		case "Pass-Through": mode = 2
		}
		go func() {
			client.SetOutputFixed(context.Background(), mode)
			sa.updateAudioSettingsUI(ui, ip)
		}()
	})
	ui.LineOutCard = makeCard("Line-Out Level", ui.LineOutSelect)
	ui.LineOutCard.Hide() // Hide by default, show in update if supported

	ui.AudioSettings = container.NewVBox(
		container.NewAdaptiveGrid(2,
			makeCard("Equalizer", container.NewGridWithColumns(2,
				container.NewVBox(widget.NewLabel("Bass"), ui.BassSlider),
				container.NewVBox(widget.NewLabel("Treble"), ui.TrebleSlider),
			)),
			makeCard("Modes", container.NewVBox(ui.LoudnessBtn, ui.NightBtn, ui.SpeechBtn)),
		),
		ui.LineOutCard,
	)

	// Grouping & Group Volume
	ui.GroupVolSlider = widget.NewSlider(0, 100)
	ui.GroupVolSlider.OnChanged = func(v float64) { go client.SetGroupVolume(context.Background(), int(v)) }
	ui.GroupMuteBtn = widget.NewButtonWithIcon("Group Mute", theme.VolumeUpIcon(), func() {
		go func() {
			ctx := context.Background()
			m, _ := client.GetGroupMute(ctx)
			client.SetGroupMute(ctx, !m)
			sa.updateGroups(ui, ip, id)
		}()
	})

	ui.GroupList = container.NewVBox()
	groupingContent := container.NewVBox(
		makeCard("Group Volume", container.NewBorder(nil, nil, ui.GroupMuteBtn, nil, ui.GroupVolSlider)),
		layout.NewSpacer(),
		makeCard("Manage Group", ui.GroupList),
	)

	ui.QueueList = container.NewVBox()
	ui.FavoritesList = container.NewVBox()
	ui.InputsList = container.NewVBox()

	refreshInputsBtn := widget.NewButtonWithIcon("Refresh Inputs", theme.ViewRefreshIcon(), func() { go sa.updateInputsUI(ui, ip) })

	ui.Tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Now Playing", theme.MediaPlayIcon(), container.NewPadded(nowPlaying)),
		container.NewTabItemWithIcon("Audio", theme.SettingsIcon(), container.NewPadded(container.NewVScroll(ui.AudioSettings))),
		container.NewTabItemWithIcon("Grouping", theme.SettingsIcon(), container.NewPadded(container.NewVScroll(groupingContent))),
		container.NewTabItemWithIcon("Inputs", theme.SettingsIcon(), container.NewBorder(refreshInputsBtn, nil, nil, nil, container.NewVScroll(ui.InputsList))),
		container.NewTabItemWithIcon("Queue", theme.ListIcon(), container.NewPadded(container.NewVScroll(ui.QueueList))),
		container.NewTabItemWithIcon("Favorites", theme.MediaPlayIcon(), container.NewPadded(container.NewVScroll(ui.FavoritesList))),
	)

	ui.Container = container.NewMax(ui.Tabs)

	// Subscribe for events
	sa.Events.Subscribe(context.Background(), ip)

	return ui
}

func (sa *SonosApp) updateInputsUI(ui *RoomUI, ip string) {
	client := sonos.NewClient(ip, 2*time.Second)
	ctx := context.Background()

	res, err := client.Browse(ctx, "AI:", 0, 50)
	if err != nil {
		slog.Error("Failed to browse AI:", "err", err, "ip", ip)
	}

	items, _ := sonos.ParseDIDLItems(res.Result)
	
	var myUUID string
	for _, m := range sa.Topology.ByIP {
		if m.IP == ip && m.UUID != "" {
			myUUID = m.UUID
			break
		}
	}

	fyne.Do(func() {
		ui.InputsList.Objects = nil
		
		found := false
		if myUUID != "" {
			tvURI := "x-sonos-htastream:" + myUUID + ":spdif"
			btn := widget.NewButtonWithIcon("TV Input", theme.MediaVideoIcon(), func() {
				go func() {
					client.PlayURI(context.Background(), tvURI, "")
					sa.updateRoomStatus(ui, ip)
				}()
			})
			ui.InputsList.Add(btn)
			found = true
		}

		for _, it := range items {
			item := it
			btn := widget.NewButtonWithIcon(it.Title, theme.MediaPlayIcon(), func() {
				go func() {
					client.PlayURI(context.Background(), item.URI, item.ResMD)
					sa.updateRoomStatus(ui, ip)
				}()
			})
			ui.InputsList.Add(btn)
			found = true
		}

		if !found {
			ui.InputsList.Add(widget.NewLabel("No physical inputs (HDMI/Line-In) found for this room."))
		}
		ui.InputsList.Refresh()
	})
}

func parseDuration(s string) time.Duration {
	parts := strings.Split(s, ":")
	if len(parts) != 3 { return 0 }
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s2, _ := strconv.Atoi(parts[2])
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s2)*time.Second
}

func (sa *SonosApp) updateRoomStatus(ui *RoomUI, ip string) {
	client := sonos.NewClient(ip, 2*time.Second)
	ctx := context.Background()
	
	pos, err := client.GetPositionInfo(ctx)
	vol, _ := client.GetVolume(ctx)
	info, _ := client.GetTransportInfo(ctx)

	fyne.Do(func() {
		if err == nil {
			uri := strings.ToLower(pos.TrackURI)
			if strings.Contains(uri, "htastream") || strings.Contains(uri, "hdmi") || strings.Contains(uri, "spdif") {
				ui.TrackLabel.SetText("TV Audio")
				ui.ArtistLabel.SetText("")
				ui.AlbumArt.Resource = theme.MediaVideoIcon()
				ui.Progress.Hide()
				ui.TimeLabel.Hide()
			} else if strings.Contains(uri, "line-in") || strings.Contains(uri, "linein") {
				ui.TrackLabel.SetText("Line-In")
				ui.ArtistLabel.SetText("")
				ui.AlbumArt.Resource = theme.SettingsIcon()
				ui.Progress.Hide()
				ui.TimeLabel.Hide()
			} else if it, ok := sonos.ParseNowPlaying(pos.TrackMeta); ok {
				ui.TrackLabel.SetText(it.Title)
				ui.ArtistLabel.SetText(it.Artist)
				if it.AlbumArtURI != "" {
					artURL := sonos.AlbumArtURL(ip, it.AlbumArtURI)
					res, err := fyne.LoadResourceFromURLString(artURL)
					if err == nil {
						ui.AlbumArt.Resource = res
					}
				}
				
				dur := parseDuration(pos.TrackDuration)
				rel := parseDuration(pos.RelTime)
				if dur > 0 {
					ui.Progress.Show()
					ui.TimeLabel.Show()
					ui.Progress.SetValue(float64(rel) / float64(dur) * 100)
					ui.TimeLabel.SetText(fmt.Sprintf("%s / %s", pos.RelTime, pos.TrackDuration))
				} else {
					ui.Progress.Hide()
					ui.TimeLabel.Hide()
				}
			} else {
				ui.TrackLabel.SetText("No Track")
				ui.ArtistLabel.SetText("")
				ui.AlbumArt.Resource = theme.BrokenImageIcon()
				ui.Progress.Hide()
				ui.TimeLabel.Hide()
			}
			ui.AlbumArt.Refresh()
		}
		
		ui.VolSlider.SetValue(float64(vol))
		
		if info.State == "PLAYING" {
			ui.PlayPauseBtn.SetIcon(theme.MediaPauseIcon())
		} else {
			ui.PlayPauseBtn.SetIcon(theme.MediaPlayIcon())
		}
	})
}

func (sa *SonosApp) updateQueue(ui *RoomUI, ip string) {
	client := sonos.NewClient(ip, 2*time.Second)
	ctx := context.Background()
	q, err := client.ListQueue(ctx, 0, 50)
	if err != nil { return }

	fyne.Do(func() {
		ui.QueueList.Objects = nil
		for _, item := range q.Items {
			it := item
			btn := widget.NewButton(fmt.Sprintf("%d. %s - %s", it.Position, it.Item.Title, it.Item.Artist), func() {
				go func() {
					client.PlayQueuePosition(context.Background(), it.Position)
					sa.updateRoomStatus(ui, ip)
				}()
			})
			ui.QueueList.Add(btn)
		}
		ui.QueueList.Refresh()
	})
}

func (sa *SonosApp) updateFavorites(ui *RoomUI, ip string) {
	client := sonos.NewClient(ip, 2*time.Second)
	ctx := context.Background()
	favs, err := client.ListFavorites(ctx, 0, 50)
	if err != nil { return }

	fyne.Do(func() {
		ui.FavoritesList.Objects = nil
		for _, f := range favs.Items {
			fav := f
			btn := widget.NewButton(f.Item.Title, func() {
				go func() {
					client.PlayFavorite(context.Background(), fav.Item)
					sa.updateRoomStatus(ui, ip)
				}()
			})
			ui.FavoritesList.Add(btn)
		}
		ui.FavoritesList.Refresh()
	})
}

func (sa *SonosApp) updateAudioSettingsUI(ui *RoomUI, ip string) {
	client := sonos.NewClient(ip, 2*time.Second)
	ctx := context.Background()
	
	bass, _ := client.GetEQ(ctx, "Bass")
	treble, _ := client.GetEQ(ctx, "Treble")
	loud, _ := client.GetLoudness(ctx)
	night, _ := client.GetNightMode(ctx)
	speech, _ := client.GetSpeechEnhancement(ctx)
	
	supportsFixed, _ := client.GetSupportsOutputFixed(ctx)
	fixedMode := -1
	if supportsFixed {
		fixedMode, _ = client.GetOutputFixed(ctx)
	}

	fyne.Do(func() {
		ui.BassSlider.SetValue(float64(bass))
		ui.TrebleSlider.SetValue(float64(treble))
		
		if loud { ui.LoudnessBtn.Importance = widget.HighImportance } else { ui.LoudnessBtn.Importance = widget.MediumImportance }
		if night { ui.NightBtn.Importance = widget.HighImportance } else { ui.NightBtn.Importance = widget.MediumImportance }
		if speech { ui.SpeechBtn.Importance = widget.HighImportance } else { ui.SpeechBtn.Importance = widget.MediumImportance }
		
		if supportsFixed {
			ui.LineOutCard.Show()
			switch fixedMode {
			case 0: ui.LineOutSelect.SetSelected("Variable")
			case 1: ui.LineOutSelect.SetSelected("Fixed")
			case 2: ui.LineOutSelect.SetSelected("Pass-Through")
			}
		} else {
			ui.LineOutCard.Hide()
		}

		ui.LoudnessBtn.Refresh()
		ui.NightBtn.Refresh()
		ui.SpeechBtn.Refresh()
		ui.AudioSettings.Refresh()
	})
}

func (sa *SonosApp) updateGroups(ui *RoomUI, ip, id string) {
	client := sonos.NewClient(ip, 2*time.Second)
	ctx := context.Background()
	
	gvol, _ := client.GetGroupVolume(ctx)
	gmute, _ := client.GetGroupMute(ctx)
	
	fyne.Do(func() {
		ui.GroupVolSlider.SetValue(float64(gvol))
		if gmute {
			ui.GroupMuteBtn.SetIcon(theme.VolumeMuteIcon())
			ui.GroupMuteBtn.Importance = widget.HighImportance
		} else {
			ui.GroupMuteBtn.SetIcon(theme.VolumeUpIcon())
			ui.GroupMuteBtn.Importance = widget.MediumImportance
		}

		ui.GroupList.Objects = nil
		for _, g := range sa.Topology.Groups {
			group := g
			btn := widget.NewButton(fmt.Sprintf("Join %s", g.Coordinator.Name), func() {
				go func() {
					client.JoinGroup(context.Background(), group.Coordinator.UUID)
					sa.Discover()
				}()
			})
			isSelf := false
			for _, m := range g.Members {
				if m.UUID == id { isSelf = true; break }
			}
			if !isSelf { ui.GroupList.Add(btn) }
		}
		
		leaveBtn := widget.NewButtonWithIcon("Leave Group (Solo)", theme.ContentRemoveIcon(), func() {
			go func() {
				client.LeaveGroup(context.Background())
				sa.Discover()
			}()
		})
		ui.GroupList.Add(widget.NewSeparator())
		ui.GroupList.Add(leaveBtn)
		ui.GroupList.Refresh()
	})
}

func (sa *SonosApp) refreshScenes() {
	metas, _ := sa.ScenesStore.List()
	fyne.Do(func() {
		sa.ScenesList.Objects = nil
		for _, m := range metas {
			meta := m
			btn := widget.NewButton(m.Name, func() { sa.applyScene(meta.Name) })
			delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				sa.ScenesStore.Delete(meta.Name)
				sa.refreshScenes()
			})
			sa.ScenesList.Add(container.NewBorder(nil, nil, nil, delBtn, btn))
		}
		sa.ScenesList.Refresh()
	})
}

func (sa *SonosApp) saveCurrentScene() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Scene Name")
	dialog.ShowForm("Save Scene", "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", entry),
	}, func(ok bool) {
		if ok && entry.Text != "" {
			go func() {
				sa.doSaveScene(entry.Text)
				sa.refreshScenes()
			}()
		}
	}, sa.Window)
}

func (sa *SonosApp) doSaveScene(name string) {
	ctx := context.Background()
	scene := scenes.Scene{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	for _, g := range sa.Topology.Groups {
		coord := g.Coordinator
		memberUUIDs := make([]string, 0, len(g.Members))
		for _, m := range g.Members {
			if !m.IsVisible { continue }
			if m.UUID != "" { memberUUIDs = append(memberUUIDs, m.UUID) }
		}
		sort.Strings(memberUUIDs)
		scene.Groups = append(scene.Groups, scenes.SceneGroup{
			ID:              g.ID,
			CoordinatorUUID: coord.UUID,
			CoordinatorName: coord.Name,
			MemberUUIDs:     memberUUIDs,
		})
	}

	seen := map[string]bool{}
	for _, g := range sa.Topology.Groups {
		for _, m := range g.Members {
			if !m.IsVisible || m.UUID == "" || seen[m.UUID] { continue }
			seen[m.UUID] = true
			c := sonos.NewClient(m.IP, 2*time.Second)
			vol, _ := c.GetVolume(ctx)
			mute, _ := c.GetMute(ctx)
			scene.Devices = append(scene.Devices, scenes.SceneDevice{
				UUID:   m.UUID,
				Name:   m.Name,
				IP:     m.IP,
				Volume: vol,
				Mute:   mute,
			})
		}
	}
	sa.ScenesStore.Put(scene)
}

func (sa *SonosApp) applyScene(name string) {
	scene, ok, _ := sa.ScenesStore.Get(name)
	if !ok { return }
	
	fyne.Do(func() { sa.StatusLabel.SetText(fmt.Sprintf("Applying scene %s...", name)) })
	
	go func() {
		ctx := context.Background()
		timeout := 2 * time.Second
		uuidToIP := map[string]string{}
		for _, m := range sa.Topology.ByIP {
			if m.UUID != "" { uuidToIP[m.UUID] = m.IP }
		}

		for _, dev := range scene.Devices {
			ip := uuidToIP[dev.UUID]
			if ip == "" { ip = dev.IP }
			if ip != "" { _ = sonos.NewClient(ip, timeout).LeaveGroup(ctx) }
		}

		for _, g := range scene.Groups {
			if g.CoordinatorUUID == "" { continue }
			for _, memberUUID := range g.MemberUUIDs {
				if memberUUID == "" || memberUUID == g.CoordinatorUUID { continue }
				memberIP := uuidToIP[memberUUID]
				if memberIP != "" { _ = sonos.NewClient(memberIP, timeout).JoinGroup(ctx, g.CoordinatorUUID) }
			}
		}

		for _, dev := range scene.Devices {
			ip := uuidToIP[dev.UUID]
			if ip == "" { ip = dev.IP }
			if ip != "" {
				c := sonos.NewClient(ip, timeout)
				_ = c.SetMute(ctx, dev.Mute)
				_ = c.SetVolume(ctx, dev.Volume)
			}
		}
		sa.Discover()
	}()
}

var _ = color.Black
