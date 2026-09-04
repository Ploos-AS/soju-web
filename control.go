package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type sojuAdminClient struct {
	SocketPath string
}

func (c sojuAdminClient) Run(ctx context.Context, words ...string) ([]string, error) {
	if strings.TrimSpace(c.SocketPath) == "" {
		return nil, errors.New("soju admin socket is not configured")
	}

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "unix", c.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to soju admin socket: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	command := quoteWords(words)
	if _, err := fmt.Fprintf(conn, "BOUNCERSERV :%s\r\n", command); err != nil {
		return nil, fmt.Errorf("write admin command: %w", err)
	}

	var output []string
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		command, params, trailing := parseIRCLine(scanner.Text())
		switch command {
		case "PRIVMSG":
			output = append(output, trailing)
		case "BOUNCERSERV":
			if len(params) > 0 && params[0] == "OK" {
				return output, nil
			}
			if trailing == "" {
				trailing = "soju rejected admin command"
			}
			return output, errors.New(trailing)
		}
	}
	if err := scanner.Err(); err != nil {
		return output, fmt.Errorf("read admin response: %w", err)
	}
	return output, errors.New("soju admin connection closed without completion response")
}

func quoteWords(words []string) string {
	var b strings.Builder
	for _, word := range words {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Quote(word))
	}
	return b.String()
}

func parseIRCLine(line string) (command string, params []string, trailing string) {
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if strings.HasPrefix(line, "@") {
		if i := strings.IndexByte(line, ' '); i >= 0 {
			line = strings.TrimLeft(line[i+1:], " ")
		}
	}
	if strings.HasPrefix(line, ":") {
		if i := strings.IndexByte(line, ' '); i >= 0 {
			line = strings.TrimLeft(line[i+1:], " ")
		}
	}

	if i := strings.Index(line, " :"); i >= 0 {
		trailing = line[i+2:]
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", nil, trailing
	}
	return strings.ToUpper(fields[0]), fields[1:], trailing
}
