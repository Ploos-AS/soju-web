package main

import "testing"

func TestMakeUserRowsEscapesUsername(t *testing.T) {
	rows := makeUserRows([]userStatus{{Username: "alice smith"}})
	if len(rows) != 1 || rows[0].NetworksURL != "/networks?user=alice+smith" {
		t.Fatalf("unexpected user navigation: %#v", rows)
	}
	if rows[0].ManageURL != "/users?manage=alice+smith#manage-user" {
		t.Fatalf("unexpected user manage URL: %q", rows[0].ManageURL)
	}
}

func TestMakeNetworkRowsPreservesContext(t *testing.T) {
	rows := makeNetworkRows("alice smith", []networkStatus{{Name: "Libera Chat"}})
	if len(rows) != 1 {
		t.Fatalf("unexpected row count: %d", len(rows))
	}
	if rows[0].ChannelsURL != "/channels?network=Libera+Chat&user=alice+smith" {
		t.Fatalf("unexpected channels URL: %q", rows[0].ChannelsURL)
	}
	if rows[0].SecurityURL != "/security?network=Libera+Chat&user=alice+smith" {
		t.Fatalf("unexpected security URL: %q", rows[0].SecurityURL)
	}
	if rows[0].ManageURL != "/networks?manage=Libera+Chat&user=alice+smith#manage-network" {
		t.Fatalf("unexpected network manage URL: %q", rows[0].ManageURL)
	}
}

func TestMakeChannelRowsPreservesContext(t *testing.T) {
	rows := makeChannelRows("alice smith", "Libera Chat", []channelStatus{{Name: "#soju test"}})
	if len(rows) != 1 {
		t.Fatalf("unexpected row count: %d", len(rows))
	}
	want := "/channels?manage=%23soju+test&network=Libera+Chat&user=alice+smith#manage-channel"
	if rows[0].ManageURL != want {
		t.Fatalf("unexpected channel manage URL: %q", rows[0].ManageURL)
	}
}
