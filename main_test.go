package main

import (
	"net/http/httptest"
	"testing"
)

func TestSignStable(t *testing.T) {
	a := &app{cfg: config{SessionSecret: []byte("01234567890123456789012345678901")}}
	if a.sign("x") != a.sign("x") { t.Fatal("signature not stable") }
	if a.sign("x") == a.sign("y") { t.Fatal("different payloads matched") }
}

func TestHealthz(t *testing.T) {
	a := &app{}
	rr := httptest.NewRecorder()
	a.healthz(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != 200 || rr.Body.String() != "ok\n" { t.Fatalf("unexpected health response: %d %q", rr.Code, rr.Body.String()) }
}
