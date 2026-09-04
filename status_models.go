package main

import (
	"fmt"
	"strconv"
	"strings"
)

type userStatus struct {
	Username    string
	Admin       bool
	Disabled    bool
	Networks    int
	MaxNetworks *int
	Raw         string
}

type networkStatus struct {
	Name      string
	Address   string
	State     string
	ConnectedAs string
	Channels  int
	Detail    string
	Raw       string
}

type channelStatus struct {
	Name     string
	State    string
	Detached bool
	Raw      string
}

func parseUserStatuses(lines []string) []userStatus {
	out := make([]userStatus, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "(") {
			continue
		}
		left, right, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		u := userStatus{Raw: line}
		left = strings.TrimSpace(left)
		if i := strings.Index(left, " ("); i >= 0 && strings.HasSuffix(left, ")") {
			u.Username = strings.TrimSpace(left[:i])
			attrs := strings.Split(strings.TrimSuffix(left[i+2:], ")"), ",")
			for _, attr := range attrs {
				switch strings.TrimSpace(attr) {
				case "admin":
					u.Admin = true
				case "disabled":
					u.Disabled = true
				}
			}
		} else {
			u.Username = left
		}
		right = strings.TrimSpace(right)
		if _, err := fmt.Sscanf(right, "%d networks", &u.Networks); err != nil {
			continue
		}
		if i := strings.Index(right, "("); i >= 0 {
			var max int
			if _, err := fmt.Sscanf(right[i:], "(%d max)", &max); err == nil {
				u.MaxNetworks = &max
			}
		}
		out = append(out, u)
	}
	return out
}

func parseNetworkStatuses(lines []string) []networkStatus {
	out := make([]networkStatus, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "No network configured") {
			continue
		}
		open := strings.LastIndex(line, " [")
		close := -1
		if open >= 0 {
			close = strings.Index(line[open:], "]")
			if close >= 0 {
				close += open
			}
		}
		if open < 0 || close < 0 {
			continue
		}
		n := networkStatus{Raw: line}
		namePart := strings.TrimSpace(line[:open])
		if i := strings.LastIndex(namePart, " ("); i >= 0 && strings.HasSuffix(namePart, ")") {
			n.Name = strings.TrimSpace(namePart[:i])
			n.Address = strings.TrimSuffix(namePart[i+2:], ")")
		} else {
			n.Name = namePart
			n.Address = namePart
		}
		statuses := strings.Split(line[open+2:close], ",")
		for _, status := range statuses {
			status = strings.TrimSpace(status)
			switch {
			case status == "connected":
				n.State = "connected"
			case strings.HasPrefix(status, "connected as "):
				n.State = "connected"
				n.ConnectedAs = strings.TrimPrefix(status, "connected as ")
			case status == "disabled":
				n.State = "disabled"
			case status == "disconnected":
				n.State = "disconnected"
			}
		}
		if close+1 < len(line) && strings.HasPrefix(line[close+1:], ": ") {
			n.Detail = strings.TrimSpace(line[close+3:])
			if n.State == "connected" {
				if fields := strings.Fields(n.Detail); len(fields) >= 2 && fields[1] == "channels" {
					n.Channels, _ = strconv.Atoi(fields[0])
				}
			}
		}
		out = append(out, n)
	}
	return out
}

func parseChannelStatuses(lines []string) []channelStatus {
	out := make([]channelStatus, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || line == "No channel configured." {
			continue
		}
		open := strings.LastIndex(line, " [")
		if open < 0 || !strings.HasSuffix(line, "]") {
			continue
		}
		c := channelStatus{Name: strings.TrimSpace(line[:open]), Raw: line}
		for _, status := range strings.Split(strings.TrimSuffix(line[open+2:], "]"), ",") {
			switch strings.TrimSpace(status) {
			case "joined":
				c.State = "joined"
			case "parted":
				c.State = "parted"
			case "disconnected":
				c.State = "disconnected"
			case "detached":
				c.Detached = true
			}
		}
		out = append(out, c)
	}
	return out
}
