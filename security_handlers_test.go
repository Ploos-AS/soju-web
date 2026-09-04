package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateCertFPRequiresExplicitConfirmation(t *testing.T) {
	a := &app{cfg: config{
		AdminUser:     "admin",
		SessionSecret: []byte("01234567890123456789012345678901"),
	}}

	for _, confirmation := range []string{"", "yes", "Generate"} {
		form := url.Values{
			"csrf":     {a.csrfToken()},
			"user":     {"alice"},
			"network":  {"Libera"},
			"key_type": {"ed25519"},
			"confirm":  {confirmation},
		}
		req := httptest.NewRequest(http.MethodPost, "/security/certfp/generate", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		a.generateCertFP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("confirmation %q: got status %d, want %d", confirmation, rr.Code, http.StatusBadRequest)
		}
	}
}
