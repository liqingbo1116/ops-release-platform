package repository

import (
	"testing"
	"time"
)

func TestEffectiveAgentStatusUsesHeartbeatTimeout(t *testing.T) {
	now := time.Now()
	recent := now.Add(-agentHeartbeatTimeout / 2)
	expired := now.Add(-agentHeartbeatTimeout - time.Second)

	tests := []struct {
		name            string
		status          string
		lastHeartbeatAt *time.Time
		want            string
	}{
		{name: "recent online heartbeat stays online", status: "ONLINE", lastHeartbeatAt: &recent, want: "ONLINE"},
		{name: "expired online heartbeat becomes offline", status: "ONLINE", lastHeartbeatAt: &expired, want: "OFFLINE"},
		{name: "missing heartbeat becomes offline", status: "ONLINE", lastHeartbeatAt: nil, want: "OFFLINE"},
		{name: "stored offline remains offline", status: "OFFLINE", lastHeartbeatAt: &recent, want: "OFFLINE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveAgentStatus(tt.status, tt.lastHeartbeatAt); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}
