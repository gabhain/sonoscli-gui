package cli

import "sonoscli-gui/internal/sonos"

// Dependency injection points for tests.
var newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) {
	return sonos.NewDefaultSMAPITokenStore()
}
