package main

import "testing"

func TestParseUserStatuses(t *testing.T) {
	rows := parseUserStatuses([]string{
		"alice (admin): 2 networks (5 max)",
		"bob (disabled): 1 networks",
	})
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
	if !rows[0].Admin || rows[0].Username != "alice" || rows[0].Networks != 2 || rows[0].MaxNetworks == nil || *rows[0].MaxNetworks != 5 {
		t.Fatalf("unexpected alice row: %+v", rows[0])
	}
	if !rows[1].Disabled || rows[1].Username != "bob" {
		t.Fatalf("unexpected bob row: %+v", rows[1])
	}
}

func TestParseNetworkStatuses(t *testing.T) {
	rows := parseNetworkStatuses([]string{
		"Libera (ircs://irc.libera.chat:6697) [connected as pgo]: 12 channels",
		"OFTC [disconnected]: dial tcp: connection refused",
		"Test [disabled]",
	})
	if len(rows) != 3 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows[0].State != "connected" || rows[0].ConnectedAs != "pgo" || rows[0].Channels != 12 {
		t.Fatalf("unexpected connected row: %+v", rows[0])
	}
	if rows[1].State != "disconnected" || rows[1].Detail == "" {
		t.Fatalf("unexpected disconnected row: %+v", rows[1])
	}
	if rows[2].State != "disabled" {
		t.Fatalf("unexpected disabled row: %+v", rows[2])
	}
}

func TestParseChannelStatuses(t *testing.T) {
	rows := parseChannelStatuses([]string{
		"#soju [joined]",
		"#ops [parted, detached]",
		"#idle [disconnected]",
	})
	if len(rows) != 3 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows[0].State != "joined" || rows[0].Detached {
		t.Fatalf("unexpected joined row: %+v", rows[0])
	}
	if rows[1].State != "parted" || !rows[1].Detached {
		t.Fatalf("unexpected detached row: %+v", rows[1])
	}
	if rows[2].State != "disconnected" {
		t.Fatalf("unexpected disconnected row: %+v", rows[2])
	}
}
