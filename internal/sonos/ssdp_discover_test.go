package sonos

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeTimeoutErr struct{}

func (fakeTimeoutErr) Error() string   { return "timeout" }
func (fakeTimeoutErr) Timeout() bool   { return true }
func (fakeTimeoutErr) Temporary() bool { return true }

type fakeSSDPConn struct {
	mu sync.Mutex

	writes []struct {
		dst  *net.UDPAddr
		data string
	}

	reads [][]byte
}

func (c *fakeSSDPConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, struct {
		dst  *net.UDPAddr
		data string
	}{dst: addr, data: string(b)})
	return len(b), nil
}

func (c *fakeSSDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.reads) == 0 {
		return 0, nil, fakeTimeoutErr{}
	}
	msg := c.reads[0]
	c.reads = c.reads[1:]
	n := copy(b, msg)
	return n, &net.UDPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 1900}, nil
}

func (c *fakeSSDPConn) SetReadDeadline(t time.Time) error { return nil }
func (c *fakeSSDPConn) Close() error                      { return nil }

func TestSSDPDiscover_SendsAndCollects(t *testing.T) {
	oldListen := ssdpListenUDP
	oldNow := ssdpNow
	t.Cleanup(func() {
		ssdpListenUDP = oldListen
		ssdpNow = oldNow
	})

	fc := &fakeSSDPConn{
		reads: [][]byte{
			[]byte("HTTP/1.1 200 OK\r\nLOCATION: http://192.168.1.10:1400/xml/device_description.xml\r\nUSN: uuid:A\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"),
			[]byte("HTTP/1.1 200 OK\r\nLOCATION: http://192.168.1.11:1400/xml/device_description.xml\r\nUSN: uuid:B\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"),
			[]byte("HTTP/1.1 200 OK\r\nLOCATION: http://192.168.1.10:1400/xml/device_description.xml\r\nUSN: uuid:A2\r\nST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"),
		},
	}
	ssdpListenUDP = func(network string, laddr *net.UDPAddr) (ssdpUDPConn, error) {
		return fc, nil
	}

	base := time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC)
	var calls int
	ssdpNow = func() time.Time {
		calls++
		return base.Add(time.Duration(calls) * 10 * time.Millisecond)
	}

	res, err := ssdpDiscover(context.Background(), 100*time.Millisecond)
	if err != nil {
		t.Fatalf("ssdpDiscover: %v", err)
	}

	fc.mu.Lock()
	writes := append([]struct {
		dst  *net.UDPAddr
		data string
	}{}, fc.writes...)
	fc.mu.Unlock()

	if len(writes) != 3 {
		t.Fatalf("expected 3 writes, got %d", len(writes))
	}
	for _, w := range writes {
		if w.dst == nil || w.dst.IP.String() != "239.255.255.250" || w.dst.Port != 1900 {
			t.Fatalf("unexpected dst: %#v", w.dst)
		}
		if !strings.Contains(w.data, "M-SEARCH * HTTP/1.1") || !strings.Contains(w.data, "ST: urn:schemas-upnp-org:device:ZonePlayer:1") {
			t.Fatalf("unexpected payload: %q", w.data)
		}
	}

	got := map[string]bool{}
	for _, r := range res {
		got[r.Location] = true
	}
	if len(got) != 2 || !got["http://192.168.1.10:1400/xml/device_description.xml"] || !got["http://192.168.1.11:1400/xml/device_description.xml"] {
		t.Fatalf("unexpected results: %#v", res)
	}
}

func TestSSDPDiscover_ContextCancelReturnsError(t *testing.T) {
	oldListen := ssdpListenUDP
	t.Cleanup(func() { ssdpListenUDP = oldListen })
	ssdpListenUDP = func(network string, laddr *net.UDPAddr) (ssdpUDPConn, error) {
		return &fakeSSDPConn{}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ssdpDiscover(ctx, 50*time.Millisecond)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestSSDPDiscover_ContextDeadlineExceededIsNotAnError(t *testing.T) {
	oldListen := ssdpListenUDP
	t.Cleanup(func() { ssdpListenUDP = oldListen })
	ssdpListenUDP = func(network string, laddr *net.UDPAddr) (ssdpUDPConn, error) {
		return &fakeSSDPConn{}, nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := ssdpDiscover(ctx, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}
