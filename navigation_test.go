package main

import "testing"

func TestMakeUserRowsEscapesUsername(t *testing.T) {
	rows := makeUserRows([]userStatus{{Username: "alice smith"}})
	if len(rows) != 1 || rows[0].NetworksURL != "/networks?user=alice+smith" {
		t.Fatalf("unexpected user navigation: %#v", rows)
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
}
