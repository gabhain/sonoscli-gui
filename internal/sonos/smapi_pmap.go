package sonos

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// fetchAndParsePresentationMap downloads and parses a Sonos SMAPI presentation map.
// It returns a map from human-facing category IDs (e.g. "tracks") to the "mappedId"
// values required for SMAPI search() calls.
func fetchAndParsePresentationMap(ctx context.Context, httpClient *http.Client, uri string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Sonos/34.7-22221 (V1)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("presentation map: http %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	return parsePresentationMapXML(raw)
}

type pmapEnvelope struct {
	SearchCategories *struct {
		Categories       []pmapCategory       `xml:"Category"`
		CustomCategories []pmapCustomCategory `xml:"CustomCategory"`
	} `xml:"SearchCategories"`
}

type pmapCategory struct {
	ID       string `xml:"id,attr"`
	MappedID string `xml:"mappedId,attr"`
	AltID    string `xml:"mappedID,attr"` // some services are inconsistent
}

type pmapCustomCategory struct {
	StringID string `xml:"stringId,attr"`
	MappedID string `xml:"mappedId,attr"`
}

func parsePresentationMapXML(raw []byte) (map[string]string, error) {
	out := map[string]string{}

	// SearchCategories is often nested somewhere inside the presentation map XML
	// (e.g. under <PresentationMap type="Search">...). We scan for it rather than
	// assuming it exists at the root.
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "SearchCategories" {
			continue
		}

		var sc struct {
			Categories       []pmapCategory       `xml:"Category"`
			CustomCategories []pmapCustomCategory `xml:"CustomCategory"`
		}
		if err := dec.DecodeElement(&sc, &se); err != nil {
			return nil, err
		}
		for _, c := range sc.Categories {
			id := strings.TrimSpace(c.ID)
			mapped := strings.TrimSpace(c.MappedID)
			if mapped == "" {
				mapped = strings.TrimSpace(c.AltID)
			}
			if id != "" && mapped != "" {
				out[id] = mapped
			}
		}
		for _, c := range sc.CustomCategories {
			id := strings.TrimSpace(c.StringID)
			mapped := strings.TrimSpace(c.MappedID)
			if id != "" && mapped != "" {
				out[id] = mapped
			}
		}
		return out, nil
	}
}
