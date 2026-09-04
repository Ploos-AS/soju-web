package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type securityPageData struct {
	CSRF        string
	User        string
	Network     string
	NetworksURL string
	ChannelsURL string
	SASLLines   []string
	CertFPLines []string
	SASLError   string
	CertFPError string
	Notice      string
}

func (a *app) securityPage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("user"))
	network := strings.TrimSpace(r.URL.Query().Get("network"))
	data := securityPageData{
		CSRF:    a.csrfToken(),
		User:    username,
		Network: network,
		Notice:  securityNoticeText(r.URL.Query().Get("ok")),
	}
	if username != "" {
		data.NetworksURL = pageURL("/networks", url.Values{"user": {username}})
	}
	if username != "" && network != "" {
		data.ChannelsURL = pageURL("/channels", url.Values{"user": {username}, "network": {network}})
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		data.SASLLines, data.SASLError = runStatus(a, ctx, username, "sasl", "status", "-network", network)
		data.CertFPLines, data.CertFPError = runStatus(a, ctx, username, "certfp", "fingerprint", "-network", network)
	}
	render(w, securityTemplate, data)
}

func runStatus(a *app, ctx context.Context, username string, words ...string) ([]string, string) {
	cmd := append([]string{"user", "run", username}, words...)
	lines, err := a.adminClient().Run(ctx, cmd...)
	if err != nil {
		return lines, err.Error()
	}
	return lines, ""
}

func (a *app) setSASLPlain(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	user, network := securityTarget(r)
	saslUser := strings.TrimSpace(r.FormValue("sasl_username"))
	password := r.FormValue("password")
	if user == "" || network == "" || saslUser == "" || password == "" {
		http.Error(w, "user, network, SASL username and password are required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	_, err := a.adminClient().Run(ctx, "user", "run", user, "sasl", "set-plain", "-network", network, saslUser, password)
	if err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	redirectSecurity(w, r, user, network, "sasl")
}

func (a *app) resetSASL(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	user, network := securityTarget(r)
	if user == "" || network == "" || r.FormValue("confirm") != "reset" {
		http.Error(w, "user, network and reset confirmation are required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	_, err := a.adminClient().Run(ctx, "user", "run", user, "sasl", "reset", "-network", network)
	if err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	redirectSecurity(w, r, user, network, "reset")
}

func (a *app) generateCertFP(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	user, network := securityTarget(r)
	keyType := strings.TrimSpace(r.FormValue("key_type"))
	if keyType != "rsa" && keyType != "ecdsa" && keyType != "ed25519" {
		keyType = "ed25519"
	}
	if user == "" || network == "" {
		http.Error(w, "user and network are required", http.StatusBadRequest)
		return
	}
	words := []string{"user", "run", user, "certfp", "generate", "-network", network, "-key-type", keyType}
	if keyType == "rsa" {
		words = append(words, "-bits", "3072")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	_, err := a.adminClient().Run(ctx, words...)
	if err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	redirectSecurity(w, r, user, network, "certfp")
}

func securityTarget(r *http.Request) (string, string) {
	return strings.TrimSpace(r.FormValue("user")), strings.TrimSpace(r.FormValue("network"))
}

func redirectSecurity(w http.ResponseWriter, r *http.Request, user, network, ok string) {
	http.Redirect(w, r, "/security?user="+url.QueryEscape(user)+"&network="+url.QueryEscape(network)+"&ok="+url.QueryEscape(ok), http.StatusSeeOther)
}

func securityNoticeText(v string) string {
	switch v {
	case "sasl":
		return "SASL PLAIN credentials saved."
	case "reset":
		return "SASL credentials reset."
	case "certfp":
		return "CertFP certificate generated."
	default:
		return ""
	}
}

const securityTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Security · soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}pre{white-space:pre-wrap;background:#111827;padding:1rem;border-radius:8px}.context a{color:#fbbf24;margin-right:.75rem}select{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563;background:#111827;color:#fff;margin:.4rem 0 1rem;width:100%}</style></head><body><main><header><div><h1>Network security</h1><nav><a href="/">Dashboard</a><a href="/users">Users</a><a href="/networks">Networks</a><a href="/channels">Channels</a><a href="/security">Security</a></nav></div><form method="post" action="/logout"><button>Sign out</button></form></header><section><form method="get"><label>soju user<input name="user" value="{{.User}}" required></label><label>Network<input name="network" value="{{.Network}}" required></label><button>Load security status</button></form></section>{{if .Notice}}<section><p class="ok">{{.Notice}}</p></section>{{end}}{{if and .User .Network}}<section><p class="muted context"><a href="{{.NetworksURL}}">← Networks</a><a href="{{.ChannelsURL}}">Channels →</a> User <code>{{.User}}</code>, network <code>{{.Network}}</code></p><h2>SASL status</h2>{{if .SASLError}}<p class="bad">{{.SASLError}}</p>{{else}}<pre>{{range .SASLLines}}{{.}}
{{end}}</pre>{{end}}<h3>Set SASL PLAIN</h3><form method="post" action="/security/sasl/set"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><input type="hidden" name="network" value="{{.Network}}"><label>SASL username<input name="sasl_username" required></label><label>Password<input type="password" name="password" autocomplete="new-password" required></label><button>Save SASL credentials</button></form><h3>Reset SASL</h3><form method="post" action="/security/sasl/reset"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><input type="hidden" name="network" value="{{.Network}}"><label>Type <code>reset</code><input name="confirm" required></label><button>Reset SASL</button></form></section><section><h2>CertFP</h2>{{if .CertFPError}}<p class="muted">{{.CertFPError}}</p>{{else}}<pre>{{range .CertFPLines}}{{.}}
{{end}}</pre>{{end}}<form method="post" action="/security/certfp/generate"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><input type="hidden" name="network" value="{{.Network}}"><label>Key type<select name="key_type"><option value="ed25519">Ed25519</option><option value="ecdsa">ECDSA</option><option value="rsa">RSA-3072</option></select></label><button>Generate certificate</button></form></section>{{end}}</main></body></html>`
