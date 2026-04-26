package sonos

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type MusicServiceAuthType string

const (
	MusicServiceAuthAnonymous  MusicServiceAuthType = "Anonymous"
	MusicServiceAuthUserID     MusicServiceAuthType = "UserId"
	MusicServiceAuthDeviceLink MusicServiceAuthType = "DeviceLink"
	MusicServiceAuthAppLink    MusicServiceAuthType = "AppLink"
)

type MusicServiceDescriptor struct {
	ID                 string               `json:"id"`
	Name               string               `json:"name"`
	Version            string               `json:"version"`
	URI                string               `json:"uri"`
	SecureURI          string               `json:"secureUri"`
	Capabilities       string               `json:"capabilities"`
	ContainerType      string               `json:"containerType"`
	Auth               MusicServiceAuthType `json:"auth"`
	ServiceType        string               `json:"serviceType"`
	PresentationMapURI string               `json:"presentationMapUri,omitempty"`
	StringsURI         string               `json:"stringsUri,omitempty"`
	ManifestURI        string               `json:"manifestUri,omitempty"`
}

func (c *Client) ListAvailableServices(ctx context.Context) ([]MusicServiceDescriptor, error) {
	resp, err := c.soapCall(ctx, controlMusicServices, urnMusicServices, "ListAvailableServices", nil)
	if err != nil {
		return nil, err
	}
	xmlPayload := strings.TrimSpace(resp["AvailableServiceDescriptorList"])
	if xmlPayload == "" {
		return nil, errors.New("missing AvailableServiceDescriptorList in response")
	}
	return parseServiceDescriptorListXML(xmlPayload)
}

type servicesEnvelope struct {
	Services []serviceElement `xml:"Service"`
}

type serviceElement struct {
	ID            string `xml:"Id,attr"`
	Name          string `xml:"Name,attr"`
	Version       string `xml:"Version,attr"`
	URI           string `xml:"Uri,attr"`
	SecureURI     string `xml:"SecureUri,attr"`
	ContainerType string `xml:"ContainerType,attr"`
	Capabilities  string `xml:"Capabilities,attr"`

	Policy *struct {
		Auth string `xml:"Auth,attr"`
	} `xml:"Policy"`

	Presentation *struct {
		Strings *struct {
			URI string `xml:"Uri,attr"`
		} `xml:"Strings"`
		PresentationMap *struct {
			URI string `xml:"Uri,attr"`
		} `xml:"PresentationMap"`
	} `xml:"Presentation"`

	Manifest *struct {
		URI string `xml:"Uri,attr"`
	} `xml:"Manifest"`
}

func parseServiceDescriptorListXML(payload string) ([]MusicServiceDescriptor, error) {
	var env servicesEnvelope
	if err := xml.Unmarshal([]byte(payload), &env); err != nil {
		return nil, err
	}
	out := make([]MusicServiceDescriptor, 0, len(env.Services))
	for _, s := range env.Services {
		id := strings.TrimSpace(s.ID)
		if id == "" {
			continue
		}
		serviceType := ""
		if n, err := strconv.Atoi(id); err == nil {
			serviceType = strconv.Itoa(n*256 + 7)
		}

		var auth MusicServiceAuthType
		if s.Policy != nil {
			auth = MusicServiceAuthType(strings.TrimSpace(s.Policy.Auth))
		}
		if auth == "" {
			auth = MusicServiceAuthAnonymous
		}

		pmapURI := ""
		stringsURI := ""
		if s.Presentation != nil {
			if s.Presentation.PresentationMap != nil {
				pmapURI = strings.TrimSpace(s.Presentation.PresentationMap.URI)
			}
			if s.Presentation.Strings != nil {
				stringsURI = strings.TrimSpace(s.Presentation.Strings.URI)
			}
		}

		manifestURI := ""
		if s.Manifest != nil {
			manifestURI = strings.TrimSpace(s.Manifest.URI)
		}

		d := MusicServiceDescriptor{
			ID:                 id,
			Name:               strings.TrimSpace(s.Name),
			Version:            strings.TrimSpace(s.Version),
			URI:                strings.TrimSpace(s.URI),
			SecureURI:          strings.TrimSpace(s.SecureURI),
			Capabilities:       strings.TrimSpace(s.Capabilities),
			ContainerType:      strings.TrimSpace(s.ContainerType),
			Auth:               auth,
			ServiceType:        serviceType,
			PresentationMapURI: pmapURI,
			StringsURI:         stringsURI,
			ManifestURI:        manifestURI,
		}

		if d.Name == "" {
			d.Name = fmt.Sprintf("Service %s", d.ID)
		}
		out = append(out, d)
	}
	return out, nil
}
