package sonos

import "testing"

func TestParsePresentationMapXML(t *testing.T) {
	raw := []byte(`
<PresentationMap>
  <SearchCategories>
    <Category id="tracks" mappedId="search:track"/>
    <Category id="albums" mappedId="search:album"/>
    <CustomCategory stringId="Blogs" mappedId="SBLG"/>
  </SearchCategories>
</PresentationMap>`)

	m, err := parsePresentationMapXML(raw)
	if err != nil {
		t.Fatalf("parsePresentationMapXML: %v", err)
	}
	if m["tracks"] != "search:track" {
		t.Fatalf("tracks mapping wrong: %q", m["tracks"])
	}
	if m["albums"] != "search:album" {
		t.Fatalf("albums mapping wrong: %q", m["albums"])
	}
	if m["Blogs"] != "SBLG" {
		t.Fatalf("custom mapping wrong: %q", m["Blogs"])
	}
}

func TestParsePresentationMapXMLNestedSearchCategories(t *testing.T) {
	raw := []byte(`
<Presentation>
  <PresentationMap type="Search">
    <Match>
      <SearchCategories>
        <Category id="artists" mappedId="artist"/>
        <Category id="tracks" mappedId="track"/>
      </SearchCategories>
    </Match>
  </PresentationMap>
</Presentation>`)

	m, err := parsePresentationMapXML(raw)
	if err != nil {
		t.Fatalf("parsePresentationMapXML: %v", err)
	}
	if m["artists"] != "artist" {
		t.Fatalf("artists mapping wrong: %q", m["artists"])
	}
	if m["tracks"] != "track" {
		t.Fatalf("tracks mapping wrong: %q", m["tracks"])
	}
}
