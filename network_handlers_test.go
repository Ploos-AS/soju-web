package main

import "testing"

func TestValidNetworkAddress(t *testing.T) {
	tests := map[string]bool{
		"irc.libera.chat":               true,
		"irc.libera.chat:6697":          true,
		"ircs://irc.libera.chat:6697":   true,
		"irc+insecure://localhost:6667": true,
		"http://example.com":            false,
		"":                              false,
		"bad host name":                 false,
		"irc.libera.chat\nBOGUS":        false,
	}
	for input, want := range tests {
		if got := validNetworkAddress(input); got != want {
			t.Fatalf("validNetworkAddress(%q)=%v, want %v", input, got, want)
		}
	}
}

func TestNetworkNoticeText(t *testing.T) {
	if got := networkNoticeText("created"); got == "" {
		t.Fatal("expected created notice")
	}
	if got := networkNoticeText("other"); got != "" {
		t.Fatalf("unexpected notice %q", got)
	}
}
