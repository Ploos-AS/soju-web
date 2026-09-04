package main

import "testing"

func TestValidChannelTarget(t *testing.T) {
	tests := []struct {
		user, network, channel string
		want                   bool
	}{
		{"alice", "Libera", "#soju", true},
		{"alice", "Libera", "&local", true},
		{"alice", "", "#soju", false},
		{"alice", "Libera", "soju", false},
		{"alice", "Libera/Other", "#soju", false},
		{"alice", "Libera", "#bad/channel", false},
		{"alice\nadmin", "Libera", "#soju", false},
	}
	for _, tc := range tests {
		if got := validChannelTarget(tc.user, tc.network, tc.channel); got != tc.want {
			t.Fatalf("validChannelTarget(%q,%q,%q)=%v, want %v", tc.user, tc.network, tc.channel, got, tc.want)
		}
	}
}

func TestChannelNoticeText(t *testing.T) {
	if channelNoticeText("created") == "" {
		t.Fatal("expected created notice")
	}
	if channelNoticeText("unknown") != "" {
		t.Fatal("expected empty notice for unknown value")
	}
}
