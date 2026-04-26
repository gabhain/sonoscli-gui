package sonos

import "testing"

func TestParseSpotifyRef(t *testing.T) {
	t.Run("canonical", func(t *testing.T) {
		ref, ok := ParseSpotifyRef("spotify:track:6NmXV4o6bmp704aPGyTVVG")
		if !ok {
			t.Fatalf("expected ok")
		}
		if ref.Kind != SpotifyTrack {
			t.Fatalf("kind: %v", ref.Kind)
		}
		if ref.EncodedID != "spotify%3atrack%3a6NmXV4o6bmp704aPGyTVVG" {
			t.Fatalf("encoded: %q", ref.EncodedID)
		}
	})

	t.Run("shareURLTrack", func(t *testing.T) {
		ref, ok := ParseSpotifyRef("https://open.spotify.com/track/6NmXV4o6bmp704aPGyTVVG?si=abc")
		if !ok {
			t.Fatalf("expected ok")
		}
		if ref.Kind != SpotifyTrack {
			t.Fatalf("kind: %v", ref.Kind)
		}
		if ref.ID != "6NmXV4o6bmp704aPGyTVVG" {
			t.Fatalf("id: %q", ref.ID)
		}
	})

	t.Run("shareURLAlbum", func(t *testing.T) {
		ref, ok := ParseSpotifyRef("https://open.spotify.com/album/4o9BvaaFDTBLFxzK70GT1E?si=lIUg4fVYRveYAovvDeac5g")
		if !ok {
			t.Fatalf("expected ok")
		}
		if ref.Kind != SpotifyAlbum {
			t.Fatalf("kind: %v", ref.Kind)
		}
		if ref.Canonical != "spotify:album:4o9BvaaFDTBLFxzK70GT1E" {
			t.Fatalf("canonical: %q", ref.Canonical)
		}
		if ref.EncodedID != "spotify%3aalbum%3a4o9BvaaFDTBLFxzK70GT1E" {
			t.Fatalf("encoded: %q", ref.EncodedID)
		}
	})

	t.Run("spotifyURLPlaylist", func(t *testing.T) {
		ref, ok := ParseSpotifyRef("spotify:user:spotify:playlist:37i9dQZF1DXcBWIGoYBM5M")
		if !ok {
			t.Fatalf("expected ok")
		}
		if ref.Kind != SpotifyPlaylist {
			t.Fatalf("kind: %v", ref.Kind)
		}
		if ref.Canonical != "spotify:playlist:37i9dQZF1DXcBWIGoYBM5M" {
			t.Fatalf("canonical: %q", ref.Canonical)
		}
	})
}
