package sonos

import "testing"

func TestParseDIDLItems_Minimal(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0"?>
<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/">
  <item id="Q:0/1" parentID="Q:0" restricted="true">
    <dc:title>Hello</dc:title>
    <upnp:class>object.item.audioItem.musicTrack</upnp:class>
    <res protocolInfo="x-rincon-playlist:*:*:*">x-sonos-spotify:spotify%3atrack%3a123</res>
  </item>
</DIDL-Lite>`

	items, err := ParseDIDLItems(xml)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected len: %d", len(items))
	}
	if items[0].Title != "Hello" {
		t.Fatalf("unexpected title: %q", items[0].Title)
	}
	if items[0].URI == "" {
		t.Fatalf("expected uri")
	}
	if items[0].Class != "object.item.audioItem.musicTrack" {
		t.Fatalf("unexpected class: %q", items[0].Class)
	}
	if items[0].ID != "Q:0/1" {
		t.Fatalf("unexpected id: %q", items[0].ID)
	}
}

func TestParseDIDLItems_ResMD(t *testing.T) {
	t.Parallel()

	didl := `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">
<item id="FV:2/1" parentID="FV:2" restricted="true">
  <dc:title>Favorite</dc:title>
  <res>x-rincon-mp3radio:http://example.com/stream</res>
  <r:resMD>&lt;DIDL-Lite&gt;&lt;item&gt;&lt;dc:title&gt;X&lt;/dc:title&gt;&lt;/item&gt;&lt;/DIDL-Lite&gt;</r:resMD>
</item>
</DIDL-Lite>`

	items, err := ParseDIDLItems(didl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ResMD == "" {
		t.Fatalf("expected resMD to be parsed")
	}
	if items[0].ResMD[0] != '<' {
		t.Fatalf("expected unescaped resMD, got: %q", items[0].ResMD)
	}
}
