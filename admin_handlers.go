package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type usersPageData struct {
	CSRF       string
	SocketPath string
	Lines      []string
	Error      string
	Notice     string
}

func (a *app) adminClient() sojuAdminClient {
	return sojuAdminClient{SocketPath: env("SOJU_ADMIN_SOCKET", "/run/soju/admin")}
}

func (a *app) csrfToken() string {
	return a.sign("csrf|" + a.cfg.AdminUser)
}

func (a *app) validCSRF(r *http.Request) bool {
	got := r.FormValue("csrf")
	want := a.csrfToken()
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (a *app) usersPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	lines, err := a.adminClient().Run(ctx, "user", "status")
	data := usersPageData{
		CSRF:       a.csrfToken(),
		SocketPath: a.adminClient().SocketPath,
		Lines:      lines,
		Notice:     noticeText(r.URL.Query().Get("ok")),
	}
	if err != nil {
		data.Error = err.Error()
	}
	render(w, usersTemplate, data)
}

func (a *app) createUser(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	if username == "" || len(password) < 8 {
		http.Error(w, "username required and password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	admin := boolValue(r.FormValue("admin"))
	enabled := boolValue(r.FormValue("enabled"))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err := a.adminClient().Run(ctx, "user", "create", "-username", username, "-password", password, "-admin", fmt.Sprint(admin), "-enabled", fmt.Sprint(enabled))
	if err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/users?ok=created", http.StatusSeeOther)
}

func (a *app) updateUser(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	admin := boolValue(r.FormValue("admin"))
	enabled := boolValue(r.FormValue("enabled"))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err := a.adminClient().Run(ctx, "user", "update", username, "-admin", fmt.Sprint(admin), "-enabled", fmt.Sprint(enabled))
	if err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/users?ok=updated", http.StatusSeeOther)
}

func (a *app) changeUserPassword(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	if username == "" || len(password) < 8 {
		http.Error(w, "username required and password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err := a.adminClient().Run(ctx, "user", "update", username, "-password", password)
	if err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/users?ok=password", http.StatusSeeOther)
}

func (a *app) prepareAdminPost(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	if !a.validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return false
	}
	return true
}

func boolValue(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "true")
}

func noticeText(v string) string {
	switch v {
	case "created":
		return "User created."
	case "updated":
		return "User updated."
	case "password":
		return "Password changed. Existing downstream sessions may be disconnected by soju."
	default:
		return ""
	}
}

const usersTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Users · soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}pre{white-space:pre-wrap;background:#111827;padding:1rem;border-radius:8px;overflow:auto}select{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563;background:#111827;color:#fff;margin:.4rem 0 1rem;width:100%}</style></head><body><main><header><div><h1>Users</h1><nav><a href="/">Dashboard</a><a href="/users">Users</a></nav></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header>{{if .Notice}}<section><p class="ok">{{.Notice}}</p></section>{{end}}<section><h2>soju user status</h2><p class="muted">Admin socket: <code>{{.SocketPath}}</code></p>{{if .Error}}<p class="bad">{{.Error}}</p>{{else}}<pre>{{range .Lines}}{{.}}
{{else}}No status lines returned.{{end}}</pre>{{end}}</section><section><h2>Create user</h2><form method="post" action="/users/create"><input type="hidden" name="csrf" value="{{.CSRF}}"><label>Username<input name="username" required></label><label>Password<input type="password" name="password" minlength="8" autocomplete="new-password" required></label><label>Administrator<select name="admin"><option value="false">No</option><option value="true">Yes</option></select></label><label>Enabled<select name="enabled"><option value="true">Yes</option><option value="false">No</option></select></label><button type="submit">Create user</button></form></section><section><h2>Update user</h2><form method="post" action="/users/update"><input type="hidden" name="csrf" value="{{.CSRF}}"><label>Username<input name="username" required></label><label>Administrator<select name="admin"><option value="false">No</option><option value="true">Yes</option></select></label><label>Enabled<select name="enabled"><option value="true">Yes</option><option value="false">No</option></select></label><button type="submit">Update user</button></form></section><section><h2>Change password</h2><form method="post" action="/users/password"><input type="hidden" name="csrf" value="{{.CSRF}}"><label>Username<input name="username" required></label><label>New password<input type="password" name="password" minlength="8" autocomplete="new-password" required></label><button type="submit">Change password</button></form></section></main></body></html>`

var _ = template.HTMLEscapeString
