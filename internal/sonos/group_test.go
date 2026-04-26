package sonos

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestJoinURI(t *testing.T) {
	if _, err := JoinURI(""); err == nil {
		t.Fatalf("expected error")
	}
	if got, err := JoinURI("RINCON_ABC1400"); err != nil || got != "x-rincon:RINCON_ABC1400" {
		t.Fatalf("JoinURI: %q err=%v", got, err)
	}
}

func TestJoinGroupAndLeaveGroup(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		switch {
		case strings.Contains(action, "AVTransport:1#SetAVTransportURI"):
			return httpResponse(200, `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`), nil
		case strings.Contains(action, "AVTransport:1#BecomeCoordinatorOfStandaloneGroup"):
			return httpResponse(200, `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:BecomeCoordinatorOfStandaloneGroupResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:BecomeCoordinatorOfStandaloneGroupResponse></s:Body></s:Envelope>`), nil
		default:
			t.Fatalf("unexpected SOAPACTION: %q", action)
			return nil, nil
		}
	})

	c := &Client{
		IP: "192.0.2.1",
		HTTP: &http.Client{
			Timeout:   time.Second,
			Transport: rt,
		},
	}

	if err := c.JoinGroup(context.Background(), ""); err == nil {
		t.Fatalf("expected error")
	}
	if err := c.JoinGroup(context.Background(), "RINCON_ABC1400"); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if err := c.LeaveGroup(context.Background()); err != nil {
		t.Fatalf("LeaveGroup: %v", err)
	}
}
