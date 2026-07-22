package api

import "testing"

func TestValidateHeartbeat(t *testing.T) {
	port := 19132
	ip := "203.0.113.10"
	domain := "relay.mc-werewolf.com"
	tests := []struct {
		name    string
		request heartbeatServerRequest
		valid   bool
	}{
		{name: "pending", request: heartbeatServerRequest{PlayerCount: 0, MaxPlayers: 10, Status: "starting", ConnectionMode: "pending"}, valid: true},
		{name: "direct ip", request: heartbeatServerRequest{PlayerCount: 2, MaxPlayers: 10, Status: "online", ConnectionMode: "direct", HostName: &ip, HostPort: &port}, valid: true},
		{name: "relay domain", request: heartbeatServerRequest{PlayerCount: 2, MaxPlayers: 10, Status: "online", ConnectionMode: "relay", HostName: &domain, HostPort: &port}, valid: true},
		{name: "missing endpoint", request: heartbeatServerRequest{PlayerCount: 0, MaxPlayers: 10, Status: "online", ConnectionMode: "direct"}, valid: false},
		{name: "too many players", request: heartbeatServerRequest{PlayerCount: 11, MaxPlayers: 10, Status: "online", ConnectionMode: "pending"}, valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if valid := validateHeartbeat(test.request) == nil; valid != test.valid {
				t.Fatalf("valid=%v, want %v", valid, test.valid)
			}
		})
	}
}

func TestValidHostName(t *testing.T) {
	for _, value := range []string{"192.168.1.2", "relay.mc-werewolf.com"} {
		if !validHostName(value) {
			t.Fatalf("expected valid hostname: %s", value)
		}
	}
	for _, value := range []string{"", "-invalid.example", "invalid name"} {
		if validHostName(value) {
			t.Fatalf("expected invalid hostname: %s", value)
		}
	}
}
