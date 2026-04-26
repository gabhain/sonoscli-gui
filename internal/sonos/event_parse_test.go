package sonos

import "testing"

func TestParseSecondTimeout(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		secs int
	}{
		{"Second-30", true, 30},
		{"second-86400", true, 86400},
		{"INFINITE", true, 0},
		{"", false, 0},
		{"Second-x", false, 0},
	}
	for _, c := range cases {
		d, ok := parseSecondTimeout(c.in)
		if ok != c.ok {
			t.Fatalf("parseSecondTimeout(%q) ok=%v want %v", c.in, ok, c.ok)
		}
		if ok && c.secs > 0 && int(d.Seconds()) != c.secs {
			t.Fatalf("parseSecondTimeout(%q) secs=%d want %d", c.in, int(d.Seconds()), c.secs)
		}
	}
}

func TestParseEventLastChangeFlatten(t *testing.T) {
	payload := []byte(`<?xml version="1.0"?>
<e:propertyset xmlns:e="urn:schemas-upnp-org:event-1-0">
  <e:property>
    <LastChange>&lt;Event xmlns=&quot;urn:schemas-upnp-org:metadata-1-0/AVT/&quot;&gt;
      &lt;InstanceID val=&quot;0&quot;&gt;
        &lt;TransportState val=&quot;PLAYING&quot;/&gt;
        &lt;CurrentTrackURI val=&quot;x-sonos-spotify:spotify%3atrack%3aabc&quot;/&gt;
      &lt;/InstanceID&gt;
    &lt;/Event&gt;</LastChange>
  </e:property>
</e:propertyset>`)

	vars, err := ParseEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["transport_state"] != "PLAYING" {
		t.Fatalf("transport_state=%q", vars["transport_state"])
	}
	if vars["current_track_uri"] == "" {
		t.Fatalf("missing current_track_uri")
	}
}

func TestParseEventChannelFlatten(t *testing.T) {
	payload := []byte(`<?xml version="1.0"?>
<e:propertyset xmlns:e="urn:schemas-upnp-org:event-1-0">
  <e:property>
    <LastChange>&lt;Event xmlns=&quot;urn:schemas-upnp-org:metadata-1-0/RCS/&quot;&gt;
      &lt;InstanceID val=&quot;0&quot;&gt;
        &lt;Volume channel=&quot;Master&quot; val=&quot;12&quot;/&gt;
        &lt;Mute channel=&quot;Master&quot; val=&quot;0&quot;/&gt;
      &lt;/InstanceID&gt;
    &lt;/Event&gt;</LastChange>
  </e:property>
</e:propertyset>`)

	vars, err := ParseEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["volume_master"] != "12" {
		t.Fatalf("volume_master=%q", vars["volume_master"])
	}
	if vars["mute_master"] != "0" {
		t.Fatalf("mute_master=%q", vars["mute_master"])
	}
}
