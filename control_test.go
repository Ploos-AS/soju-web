package main

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuoteWords(t *testing.T) {
	got := quoteWords([]string{"user", "create", "-username", "alice bob"})
	if got != `"user" "create" "-username" "alice bob"` {
		t.Fatalf("unexpected command: %q", got)
	}
}

func TestParseIRCLine(t *testing.T) {
	cmd, params, trailing := parseIRCLine(":soju PRIVMSG * :hello world\r")
	if cmd != "PRIVMSG" || len(params) != 1 || params[0] != "*" || trailing != "hello world" {
		t.Fatalf("unexpected parse: %q %#v %q", cmd, params, trailing)
	}
}

func TestAdminClientRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			done <- err
			return
		}
		if !strings.Contains(line, `"user" "status"`) {
			done <- &testError{"unexpected request: " + line}
			return
		}
		_, _ = conn.Write([]byte(":soju PRIVMSG * :alice enabled admin\r\nBOUNCERSERV OK :done\r\n"))
		done <- nil
	}()

	out, err := (sojuAdminClient{SocketPath: path}).Run(context.Background(), "user", "status")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "alice enabled admin" {
		t.Fatalf("unexpected output: %#v", out)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
}

type testError struct{ s string }

func (e *testError) Error() string { return e.s }
