package main

import "testing"

func TestParseServerStats(t *testing.T) {
	a, s, d, u, n, c, err := parseServerStats("2/3 users, 4 downstreams, 5 upstreams, 6 networks, 7 channels")
	if err != nil {
		t.Fatal(err)
	}
	if a != 2 || s != 3 || d != 4 || u != 5 || n != 6 || c != 7 {
		t.Fatalf("unexpected stats: %d %d %d %d %d %d", a, s, d, u, n, c)
	}
}

func TestSummarizeUsers(t *testing.T) {
	s := summarizeUsers([]string{"alice (admin): 2 networks", "bob (disabled): 1 networks", "(3 more users omitted)"})
	if s.Admins != 1 || s.Disabled != 1 || len(s.Lines) != 2 {
		t.Fatalf("unexpected summary: %#v", s)
	}
}

func TestAttentionMessages(t *testing.T) {
	data := dashboardData{Status: "online", AdminStatus: "online", ActiveUsers: 1, StoredUsers: 2, Upstreams: 1, Networks: 3, DisabledUsers: 1}
	if got := len(attentionMessages(data)); got != 3 {
		t.Fatalf("expected 3 attention messages, got %d", got)
	}
}
