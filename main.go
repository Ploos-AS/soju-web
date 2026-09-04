package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type config struct {
	ListenAddr    string
	AdminUser     string
	AdminPassword string
	SessionSecret []byte
	SojuAddress   string
	CookieSecure  bool
}

type app struct {
	cfg config
}

type dashboardData struct {
	Status        string
	Latency       string
	SojuAddress   string
	AdminStatus   string
	StatsError    string
	ActiveUsers   int
	StoredUsers   int
	Downstreams   int
	Upstreams     int
	Networks      int
	Channels      int
	DisabledUsers int
	AdminUsers    int
	UserLines     []string
	Attention     []string
	RawStatsLine  string
	Version       string
	Revision      string
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	a := &app{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /login", a.loginPage)
	mux.HandleFunc("POST /login", a.login)
	mux.HandleFunc("POST /logout", a.logout)
	mux.HandleFunc("GET /", a.requireAuth(a.dashboard))
	mux.HandleFunc("GET /users", a.requireAuth(a.usersPage))
	mux.HandleFunc("POST /users/create", a.requireAuth(a.createUser))
	mux.HandleFunc("POST /users/update", a.requireAuth(a.updateUser))
	mux.HandleFunc("POST /users/password", a.requireAuth(a.changeUserPassword))
	mux.HandleFunc("GET /users/delete", a.requireAuth(a.deleteUserPage))
	mux.HandleFunc("POST /users/delete", a.requireAuth(a.deleteUser))
	mux.HandleFunc("GET /networks", a.requireAuth(a.networksPage))
	mux.HandleFunc("POST /networks/create", a.requireAuth(a.createNetwork))
	mux.HandleFunc("POST /networks/update", a.requireAuth(a.updateNetwork))
	mux.HandleFunc("POST /networks/delete", a.requireAuth(a.deleteNetwork))
	mux.HandleFunc("GET /channels", a.requireAuth(a.channelsPage))
	mux.HandleFunc("POST /channels/create", a.requireAuth(a.createChannel))
	mux.HandleFunc("POST /channels/update", a.requireAuth(a.updateChannel))
	mux.HandleFunc("POST /channels/delete", a.requireAuth(a.deleteChannel))
	mux.HandleFunc("GET /security", a.requireAuth(a.securityPage))
	mux.HandleFunc("POST /security/sasl/set", a.requireAuth(a.setSASLPlain))
	mux.HandleFunc("POST /security/sasl/reset", a.requireAuth(a.resetSASL))
	mux.HandleFunc("POST /security/certfp/generate", a.requireAuth(a.generateCertFP))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("soju-web %s (%s) listening on %s; soju backend=%s", version, revision, cfg.ListenAddr, cfg.SojuAddress)
	log.Fatal(srv.ListenAndServe())
}

func loadConfig() (config, error) {
	cfg := config{
		ListenAddr:    env("SOJU_WEB_LISTEN", ":8080"),
		AdminUser:     env("SOJU_WEB_ADMIN_USER", "admin"),
		AdminPassword: os.Getenv("SOJU_WEB_ADMIN_PASSWORD"),
		SojuAddress:   env("SOJU_ADDRESS", "soju:6667"),
		CookieSecure:  strings.EqualFold(env("SOJU_WEB_COOKIE_SECURE", "false"), "true"),
	}
	if len(cfg.AdminPassword) < 12 {
		return config{}, fmt.Errorf("SOJU_WEB_ADMIN_PASSWORD must be at least 12 characters")
	}
	secret := os.Getenv("SOJU_WEB_SESSION_SECRET")
	if secret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return config{}, fmt.Errorf("generate session secret: %w", err)
		}
		cfg.SessionSecret = buf
		log.Printf("warning: SOJU_WEB_SESSION_SECRET is unset; sessions will be invalid after restart")
	} else {
		if len(secret) < 32 {
			return config{}, fmt.Errorf("SOJU_WEB_SESSION_SECRET must be at least 32 characters")
		}
		cfg.SessionSecret = []byte(secret)
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (a *app) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (a *app) loginPage(w http.ResponseWriter, r *http.Request) {
	if a.authenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	render(w, loginTemplate, map[string]any{"Error": r.URL.Query().Get("error") != ""})
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	userOK := subtle.ConstantTimeCompare([]byte(r.FormValue("username")), []byte(a.cfg.AdminUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(a.cfg.AdminPassword)) == 1
	if !userOK || !passOK {
		time.Sleep(150 * time.Millisecond)
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	expires := time.Now().Add(12 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", a.cfg.AdminUser, expires)
	value := payload + "|" + a.sign(payload)
	http.SetCookie(w, &http.Cookie{
		Name:     "soju_web_session",
		Value:    base64.RawURLEncoding.EncodeToString([]byte(value)),
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   12 * 60 * 60,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "soju_web_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *app) authenticated(r *http.Request) bool {
	c, err := r.Cookie("soju_web_session")
	if err != nil {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || parts[0] != a.cfg.AdminUser {
		return false
	}
	var expires int64
	if _, err := fmt.Sscanf(parts[1], "%d", &expires); err != nil || time.Now().Unix() > expires {
		return false
	}
	payload := parts[0] + "|" + parts[1]
	return hmac.Equal([]byte(a.sign(payload)), []byte(parts[2]))
}

func (a *app) sign(value string) string {
	m := hmac.New(sha256.New, a.cfg.SessionSecret)
	_, _ = m.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (a *app) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *app) dashboard(w http.ResponseWriter, r *http.Request) {
	data := dashboardData{
		Status:      "offline",
		Latency:     "—",
		SojuAddress: a.cfg.SojuAddress,
		AdminStatus: "offline",
		Version:     version,
		Revision:    revision,
	}

	started := time.Now()
	if conn, err := net.DialTimeout("tcp", a.cfg.SojuAddress, 2*time.Second); err == nil {
		data.Status = "online"
		data.Latency = time.Since(started).Round(time.Millisecond).String()
		_ = conn.Close()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	lines, err := a.adminClient().Run(ctx, "server", "status")
	if err != nil {
		data.StatsError = err.Error()
	} else {
		data.AdminStatus = "online"
		if len(lines) > 0 {
			data.RawStatsLine = lines[0]
			data.ActiveUsers, data.StoredUsers, data.Downstreams, data.Upstreams, data.Networks, data.Channels, err = parseServerStats(lines[0])
			if err != nil {
				data.StatsError = "unexpected server status format: " + lines[0]
			}
		}

		userLines, userErr := a.adminClient().Run(ctx, "user", "status")
		if userErr == nil {
			s := summarizeUsers(userLines)
			data.DisabledUsers = s.Disabled
			data.AdminUsers = s.Admins
			data.UserLines = s.Lines
		} else if data.StatsError == "" {
			data.StatsError = "user status: " + userErr.Error()
		}
	}

	data.Attention = attentionMessages(data)
	render(w, dashboardTemplate, data)
}

func render(w http.ResponseWriter, src string, data any) {
	t := template.Must(template.New("page").Parse(src))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

const baseCSS = `body{font-family:system-ui,sans-serif;background:#111827;color:#e5e7eb;margin:0}main{max-width:1000px;margin:6vh auto;padding:2rem}section{background:#1f2937;border:1px solid #374151;border-radius:14px;padding:1.5rem;margin-bottom:1rem}input,button{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563}input{width:100%;box-sizing:border-box;background:#111827;color:#fff;margin:.4rem 0 1rem}button{background:#f59e0b;color:#111827;font-weight:700;cursor:pointer}.ok{color:#34d399}.bad{color:#f87171}.warn{color:#fbbf24}.muted{color:#9ca3af}header{display:flex;justify-content:space-between;align-items:center}.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:1rem}.stat{background:#111827;border-radius:10px;padding:1rem}.stat strong{font-size:1.8rem;display:block}pre{white-space:pre-wrap;background:#111827;padding:1rem;border-radius:8px;overflow:auto}`

const loginTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>soju-web login</title><style>` + baseCSS + `</style></head><body><main><section><h1>soju-web</h1><p class="muted">Administrative web interface for soju.</p>{{if .Error}}<p class="bad">Invalid credentials.</p>{{end}}<form method="post" action="/login"><label>Username<input name="username" autocomplete="username" required></label><label>Password<input type="password" name="password" autocomplete="current-password" required></label><button type="submit">Sign in</button></form></section></main></body></html>`

const dashboardTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}</style></head><body><main><header><div><h1>soju-web</h1><p class="muted">Go WebAdmin for soju · {{.Version}} · <code>{{.Revision}}</code></p><nav><a href="/">Dashboard</a><a href="/users">Users</a><a href="/networks">Networks</a><a href="/channels">Channels</a><a href="/security">Security</a></nav></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header><section><h2>Health</h2><div class="stats"><div class="stat"><span>IRC listener</span><strong class="{{if eq .Status "online"}}ok{{else}}bad{{end}}">{{.Status}}</strong><small>{{.Latency}}</small></div><div class="stat"><span>Admin socket</span><strong class="{{if eq .AdminStatus "online"}}ok{{else}}bad{{end}}">{{.AdminStatus}}</strong></div></div><p class="muted">Backend: <code>{{.SojuAddress}}</code></p></section><section><h2>Server statistics</h2>{{if .StatsError}}<p class="bad">{{.StatsError}}</p>{{end}}<div class="stats"><div class="stat"><span>Users active / stored</span><strong>{{.ActiveUsers}} / {{.StoredUsers}}</strong></div><div class="stat"><span>Admins / disabled</span><strong>{{.AdminUsers}} / {{.DisabledUsers}}</strong></div><div class="stat"><span>Downstreams</span><strong>{{.Downstreams}}</strong></div><div class="stat"><span>Upstreams</span><strong>{{.Upstreams}}</strong></div><div class="stat"><span>Networks</span><strong>{{.Networks}}</strong></div><div class="stat"><span>Channels</span><strong>{{.Channels}}</strong></div></div>{{if .RawStatsLine}}<p class="muted"><code>{{.RawStatsLine}}</code></p>{{end}}</section><section><h2>Attention needed</h2>{{if .Attention}}<ul>{{range .Attention}}<li class="warn">{{.}}</li>{{end}}</ul>{{else}}<p class="ok">No operational warnings from the available soju status data.</p>{{end}}</section><section><h2>User overview</h2>{{if .UserLines}}<pre>{{range .UserLines}}{{.}}
{{end}}</pre>{{else}}<p class="muted">No user status lines available.</p>{{end}}</section><section><h2>M5</h2><p>The dashboard now combines listener health, admin-socket health, authoritative soju server statistics, user state and derived operational warnings. IRC chat remains intentionally out of scope.</p></section></main></body></html>`