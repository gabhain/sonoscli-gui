package sonos

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"
)

type ssdpResult struct {
	Location string
	USN      string
	ST       string
	Server   string
}

type ssdpUDPConn interface {
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	SetReadDeadline(t time.Time) error
	LocalAddr() net.Addr
	Close() error
}

var ssdpListenUDP = func(network string, laddr *net.UDPAddr) (ssdpUDPConn, error) {
	return net.ListenUDP(network, laddr)
}

var ssdpNow = time.Now

func ssdpDiscover(ctx context.Context, timeout time.Duration) ([]ssdpResult, error) {
	// SSDP M-SEARCH for Sonos ZonePlayer devices.
	payload := strings.Join([]string{
		"M-SEARCH * HTTP/1.1",
		"HOST: 239.255.255.250:1900",
		`MAN: "ssdp:discover"`,
		"MX: 1",
		"ST: urn:schemas-upnp-org:device:ZonePlayer:1",
		"", "",
	}, "\r\n")

	conn, err := ssdpListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if laddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		slog.Debug("ssdp: bound locally", "addr", laddr.String())
	}

	dst := &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900}

	// Try to send M-SEARCH on all available multicast-capable interfaces.
	// This helps on macOS where the default route might not include multicast.
	ifaces, _ := net.Interfaces()
	sentCount := 0
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		
		// Get addresses for this interface
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			// Bind to specific interface IP and send
			localConn, err := ssdpListenUDP("udp4", &net.UDPAddr{IP: ipNet.IP, Port: 0})
			if err != nil {
				continue
			}
			
			for i := 0; i < 2; i++ {
				if _, err := localConn.WriteToUDP([]byte(payload), dst); err != nil {
					slog.Debug("ssdp: write error on interface", "iface", iface.Name, "ip", ipNet.IP.String(), "err", err)
				} else {
					sentCount++
				}
			}
			localConn.Close()
		}
	}

	// Fallback: if no specific interface succeeded, try the default connection.
	if sentCount == 0 {
		for i := 0; i < 3; i++ {
			if _, err := conn.WriteToUDP([]byte(payload), dst); err != nil {
				slog.Debug("ssdp: fallback write error", "err", err)
			}
		}
	}
	slog.Debug("ssdp: sent M-SEARCH", "count", sentCount, "dst", dst.String())

	deadline := ssdpNow().Add(timeout)
	byLocation := map[string]ssdpResult{}

	buf := make([]byte, 64*1024)
Loop:
	for {
		if ssdpNow().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			// Treat DeadlineExceeded like a normal timeout so callers can fall back.
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				break Loop
			}
			return nil, ctx.Err()
		default:
		}

		_ = conn.SetReadDeadline(ssdpNow().Add(200 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			slog.Debug("ssdp: read error", "err", err)
			continue
		}
		msg := buf[:n]
		res, ok := parseSSDPResponse(msg)
		if !ok || res.Location == "" {
			continue
		}
		slog.Debug("ssdp: response", "location", res.Location, "usn", res.USN, "server", res.Server)
		byLocation[res.Location] = res
	}

	out := make([]ssdpResult, 0, len(byLocation))
	for _, v := range byLocation {
		out = append(out, v)
	}
	return out, nil
}

func parseSSDPResponse(b []byte) (ssdpResult, bool) {
	// SSDP responses are HTTP-like with CRLF line endings.
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Split(bufio.ScanLines)

	// First line should be "HTTP/1.1 200 OK"
	if !s.Scan() {
		return ssdpResult{}, false
	}
	first := strings.TrimSpace(s.Text())
	if !strings.HasPrefix(first, "HTTP/") {
		return ssdpResult{}, false
	}

	headers := map[string]string{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}

	return ssdpResult{
		Location: headers["location"],
		USN:      headers["usn"],
		ST:       headers["st"],
		Server:   headers["server"],
	}, true
}

func hostToIP(location string) (string, error) {
	u, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("location host missing: %q", location)
	}
	return host, nil
}
