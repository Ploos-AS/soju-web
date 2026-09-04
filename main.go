package main

import (
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

var (
	version  = "dev"
	revision = "unknown"
)

type config struct {
	ListenAddr    string
	AdminUser     string
	AdminPassword string
	SessionSecret []byte
	SojuAddress   string
	CookieSecure  bool
}

type app struct{ cfg config }

type pageData struct {
	User           string
	SojuAddress    string
	SojuReachable  bool
	SojuLatency    string
	SojuError      string
	AdminReachable bool
	AdminError     string
	Stats          sojuStats
	StatsError     string
	Users          userOverview
	UsersError     string
	Alerts         []string
	Version        string
	Revision       string
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
	exp := time.Now().Add(12 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", a.cfg.AdminUser, exp)
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + a.sign(payload)
	http.SetCookie(w, &http.Cookie{Name: "soju_web_session", Value: value, Path: "/", HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: 12 * 60 * 60})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "soju_web_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *app) authenticated(r *http.Request) bool {
	c, err := r.Cookie("soju_web_session")
	if err != nil {
		return false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	payload := string(decoded)
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(a.sign(payload))) != 1 {
		return false
	}
	var user string
	var exp int64
	if _, err := fmt.Sscanf(payload, "%[^|]|%d", &user, &exp); err != nil {
		pieces := strings.Split(payload, "|")
		if len(pieces) != 2 {
			return false
		}
		user = pieces[0]
		if _, err := fmt.Sscan(pieces[1], &exp); err != nil {
			return false
		}
	}
	return user == a.cfg.AdminUser && time.Now().Unix() <= exp
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
	data := pageData{User: a.cfg.AdminUser, SojuAddress: a.cfg.SojuAddress, Version: version, Revision: revision}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", a.cfg.SojuAddress, 2*time.Second)
	if err != nil {
		data.SojuError = err.Error()
	} else {
		data.SojuReachable = true
		data.SojuLatency = time.Since(start).Round(time.Millisecond).String()
		_ = conn.Close()
	}
	ctx, cancel := contextWithTimeout(r, 4*time.Second)
	defer cancel()
	client := a.adminClient()
	serverLines, serverErr := client.Run(ctx, "server", "status")
	if serverErr != nil {
		data.AdminError = serverErr.Error()
		data.StatsError = serverErr.Error()
	} else {
		data.AdminReachable = true
		data.Stats = parseSojuStats(serverLines)
	}
	userLines, userErr := client.Run(ctx, "user", "status")
	if userErr != nil {
		data.UsersError = userErr.Error()
	} else {
		data.Users = parseUserOverview(userLines)
	}
	data.Alerts = buildAlerts(data.SojuReachable, data.AdminReachable, data.Stats, data.Users)
	render(w, dashboardTemplate, data)
}

func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}

func (a *app) sign(value string) string {
	mac := hmac.New(sha256.New, a.cfg.SessionSecret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func render(w http.ResponseWriter, body string, data any) {
	t, err := template.New("page").Parse(body)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		log.Printf("template execute: %v", err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

const baseCSS = `body{font:16px system-ui;margin:0;background:#0b1220;color:#e5e7eb}main{max-width:1100px;margin:auto;padding:2rem}header{display:flex;justify-content:space-between;gap:1rem;align-items:center}section{background:#182235;padding:1.25rem;border-radius:12px;margin:1rem 0}input,button{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563;background:#111827;color:#fff}input{width:100%;box-sizing:border-box;margin:.4rem 0 1rem}button{cursor:pointer;background:#1d4ed8}code{color:#fbbf24}.ok{color:#86efac}.bad{color:#fca5a5}.muted{color:#9ca3af}a{color:#93c5fd}`

const loginTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>soju-web</title><style>` + baseCSS + `main{max-width:420px;margin:8vh auto}</style></head><body><main><h1>soju-web</h1><section><form method="post" action="/login"><label>Username<input name="username" autocomplete="username" required></label><label>Password<input type="password" name="password" autocomplete="current-password" required></label>{{if .Error}}<p class="bad">Invalid credentials.</p>{{end}}<button type="submit">Sign in</button></form></section></main></body></html>`

const dashboardTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>soju-web dashboard</title><style>` + baseCSS + `.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:.8rem}.stat{background:#111827;padding:1rem;border-radius:8px}.stat strong{display:block;font-size:1.5rem}nav a{color:#fbbf24;margin-right:1rem}ul{line-height:1.6}</style></head><body><main><header><div><h1>soju-web</h1><nav><a href="/">Dashboard</a><a href="/users">Users</a><a href="/networks">Networks</a><a href="/channels">Channels</a><a href="/security">Security</a></nav><p>Signed in as <strong>{{.User}}</strong></p></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header><section><h2>Health</h2><p>IRC listener <code>{{.SojuAddress}}</code>: {{if .SojuReachable}}<span class="ok">reachable ({{.SojuLatency}})</span>{{else}}<span class="bad">unreachable</span>{{end}}</p>{{if .SojuError}}<p class="muted">{{.SojuError}}</p>{{end}}<p>Admin socket: {{if .AdminReachable}}<span class="ok">reachable</span>{{else}}<span class="bad">unreachable</span>{{end}}</p>{{if .AdminError}}<p class="muted">{{.AdminError}}</p>{{end}}</section><section><h2>soju status</h2>{{if .StatsError}}<p class="bad">{{.StatsError}}</p>{{else}}<div class="grid"><div class="stat"><span>Active users</span><strong>{{.Stats.ActiveUsers}}</strong></div><div class="stat"><span>Stored users</span><strong>{{.Stats.StoredUsers}}</strong></div><div class="stat"><span>Downstreams</span><strong>{{.Stats.Downstreams}}</strong></div><div class="stat"><span>Upstreams</span><strong>{{.Stats.Upstreams}}</strong></div><div class="stat"><span>Networks</span><strong>{{.Stats.Networks}}</strong></div><div class="stat"><span>Channels</span><strong>{{.Stats.Channels}}</strong></div></div>{{end}}</section><section><h2>User overview</h2>{{if .UsersError}}<p class="bad">{{.UsersError}}</p>{{else}}<div class="grid"><div class="stat"><span>Visible users</span><strong>{{.Users.Total}}</strong></div><div class="stat"><span>Admins</span><strong>{{.Users.Admins}}</strong></div><div class="stat"><span>Disabled</span><strong>{{.Users.Disabled}}</strong></div></div>{{end}}</section><section><h2>Attention needed</h2>{{if .Alerts}}<ul>{{range .Alerts}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="ok">No status-derived warnings.</p>{{end}}</section><section><h2>Build</h2><p>soju-web <code>{{.Version}}</code> · revision <code>{{.Revision}}</code></p><p class="muted">No IRC chat is exposed by this WebAdmin.</p></section></main></body></html>`
