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
	mux.HandleFunc("GET /networks", a.requireAuth(a.networksPage))
	mux.HandleFunc("POST /networks/create", a.requireAuth(a.createNetwork))
	mux.HandleFunc("POST /networks/update", a.requireAuth(a.updateNetwork))
	mux.HandleFunc("POST /networks/delete", a.requireAuth(a.deleteNetwork))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("soju-web listening on %s; soju backend=%s", cfg.ListenAddr, cfg.SojuAddress)
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
	http.SetCookie(w, &http.Cookie{Name: "soju_web_session", Value: "", Path: "/", HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
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
	expected := a.sign(payload)
	return hmac.Equal([]byte(expected), []byte(parts[2]))
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

func (a *app) dashboard(w http.ResponseWriter, _ *http.Request) {
	status := "offline"
	latency := "—"
	started := time.Now()
	conn, err := net.DialTimeout("tcp", a.cfg.SojuAddress, 2*time.Second)
	if err == nil {
		status = "online"
		latency = time.Since(started).Round(time.Millisecond).String()
		_ = conn.Close()
	}
	render(w, dashboardTemplate, map[string]any{
		"Status":      status,
		"Latency":     latency,
		"SojuAddress": a.cfg.SojuAddress,
	})
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

const baseCSS = `body{font-family:system-ui,sans-serif;background:#111827;color:#e5e7eb;margin:0}main{max-width:900px;margin:8vh auto;padding:2rem}section{background:#1f2937;border:1px solid #374151;border-radius:14px;padding:1.5rem;margin-bottom:1rem}input,button{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563}input{width:100%;box-sizing:border-box;background:#111827;color:#fff;margin:.4rem 0 1rem}button{background:#f59e0b;color:#111827;font-weight:700;cursor:pointer}.ok{color:#34d399}.bad{color:#f87171}.muted{color:#9ca3af}header{display:flex;justify-content:space-between;align-items:center}`

const loginTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>soju-web login</title><style>` + baseCSS + `</style></head><body><main><section><h1>soju-web</h1><p class="muted">Administrative web interface for soju.</p>{{if .Error}}<p class="bad">Invalid credentials.</p>{{end}}<form method="post" action="/login"><label>Username<input name="username" autocomplete="username" required></label><label>Password<input type="password" name="password" autocomplete="current-password" required></label><button type="submit">Sign in</button></form></section></main></body></html>`

const dashboardTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}</style></head><body><main><header><div><h1>soju-web</h1><p class="muted">Go WebAdmin for soju</p><nav><a href="/">Dashboard</a><a href="/users">Users</a><a href="/networks">Networks</a></nav></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header><section><h2>soju backend</h2><p>Address: <code>{{.SojuAddress}}</code></p><p>Status: {{if eq .Status "online"}}<strong class="ok">online</strong>{{else}}<strong class="bad">offline</strong>{{end}}</p><p>TCP latency: {{.Latency}}</p></section><section><h2>M2</h2><p>User and IRC network administration are available through soju's Unix admin interface. No Docker socket and no direct database access are required.</p></section></main></body></html>`
