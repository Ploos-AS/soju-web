package main

import (
	"fmt"
	"strings"
)

var (
	version  = "dev"
	revision = "unknown"
)

type userSummary struct {
	Disabled int
	Admins   int
	Lines    []string
}

func parseServerStats(line string) (activeUsers, storedUsers, downstreams, upstreams, networks, channels int, err error) {
	_, err = fmt.Sscanf(
		line,
		"%d/%d users, %d downstreams, %d upstreams, %d networks, %d channels",
		&activeUsers,
		&storedUsers,
		&downstreams,
		&upstreams,
		&networks,
		&channels,
	)
	return
}

func summarizeUsers(lines []string) userSummary {
	var s userSummary
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "(") {
			continue
		}
		s.Lines = append(s.Lines, line)
		prefix := line
		if i := strings.IndexByte(prefix, ':'); i >= 0 {
			prefix = prefix[:i]
		}
		if strings.Contains(prefix, "disabled") {
			s.Disabled++
		}
		if strings.Contains(prefix, "admin") {
			s.Admins++
		}
	}
	return s
}

func attentionMessages(data dashboardData) []string {
	var out []string
	if data.Status != "online" {
		out = append(out, "IRC listener is not reachable from soju-web.")
	}
	if data.AdminStatus != "online" {
		out = append(out, "soju admin socket is unavailable; administrative data may be stale or missing.")
	}
	if data.StoredUsers > data.ActiveUsers {
		out = append(out, fmt.Sprintf("%d stored user(s) are not currently active.", data.StoredUsers-data.ActiveUsers))
	}
	if data.Networks > data.Upstreams {
		out = append(out, fmt.Sprintf("%d configured network(s) are not currently connected upstream; this includes disabled networks.", data.Networks-data.Upstreams))
	}
	if data.DisabledUsers > 0 {
		out = append(out, fmt.Sprintf("%d user account(s) are disabled.", data.DisabledUsers))
	}
	return out
}
