package cli

import "testing"

func TestListenIPForRemote_Localhost(t *testing.T) {
	ip, err := listenIPForRemote("127.0.0.1")
	if err != nil {
		t.Fatalf("listenIPForRemote: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Fatalf("unexpected ip: %q", ip)
	}
}
