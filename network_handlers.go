package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type networksPageData struct {
	CSRF       string
	SocketPath string
	User       string
	Lines      []string
	Error      string
	Notice     string
}

func (a *app) networksPage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("user"))
	data := networksPageData{CSRF: a.csrfToken(), SocketPath: a.adminClient().SocketPath, User: username, Notice: networkNoticeText(r.URL.Query().Get("ok"))}
	if username != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		lines, err := a.adminClient().Run(ctx, "user", "run", username, "network", "status")
		data.Lines = lines
		if err != nil {
			data.Error = err.Error()
		}
	}
	render(w, networksTemplate, data)
}

func (a *app) createNetwork(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("user"))
	addr := strings.TrimSpace(r.FormValue("addr"))
	if username == "" || !validNetworkAddress(addr) {
		http.Error(w, "user and a valid network address are required", http.StatusBadRequest)
		return
	}
	words := []string{"user", "run", username, "network", "create", "-addr", addr}
	words = appendNetworkOptions(words, r)
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, words...); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/networks?user="+url.QueryEscape(username)+"&ok=created", http.StatusSeeOther)
}

func (a *app) updateNetwork(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("user"))
	name := strings.TrimSpace(r.FormValue("name"))
	if username == "" || name == "" {
		http.Error(w, "user and network name are required", http.StatusBadRequest)
		return
	}
	words := []string{"user", "run", username, "network", "update", name}
	if addr := strings.TrimSpace(r.FormValue("addr")); addr != "" {
		if !validNetworkAddress(addr) {
			http.Error(w, "invalid network address", http.StatusBadRequest)
			return
		}
		words = append(words, "-addr", addr)
	}
	words = appendNetworkOptions(words, r)
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, words...); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/networks?user="+url.QueryEscape(username)+"&ok=updated", http.StatusSeeOther)
}

func (a *app) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("user"))
	name := strings.TrimSpace(r.FormValue("name"))
	if username == "" || name == "" || r.FormValue("confirm") != "delete" {
		http.Error(w, "user, network name and delete confirmation are required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, "user", "run", username, "network", "delete", name); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/networks?user="+url.QueryEscape(username)+"&ok=deleted", http.StatusSeeOther)
}

func appendNetworkOptions(words []string, r *http.Request) []string {
	for _, field := range []struct{ form, flag string }{{"network_name", "-name"}, {"irc_username", "-username"}, {"nick", "-nick"}, {"realname", "-realname"}} {
		if v := strings.TrimSpace(r.FormValue(field.form)); v != "" {
			words = append(words, field.flag, v)
		}
	}
	if v := strings.TrimSpace(r.FormValue("enabled")); v != "" {
		words = append(words, "-enabled", fmt.Sprint(boolValue(v)))
	}
	return words
}

func validNetworkAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" || strings.ContainsAny(addr, "\r\n\x00") {
		return false
	}
	if !strings.Contains(addr, "://") {
		return !strings.ContainsAny(addr, " \t")
	}
	u, err := url.Parse(addr)
	return err == nil && u.Host != "" && (u.Scheme == "ircs" || u.Scheme == "irc+insecure")
}

func networkNoticeText(v string) string {
	switch v {
	case "created":
		return "Network created."
	case "updated":
		return "Network updated."
	case "deleted":
		return "Network deleted."
	default:
		return ""
	}
}

const networksTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Networks · soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}pre{white-space:pre-wrap;background:#111827;padding:1rem;border-radius:8px;overflow:auto}select{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563;background:#111827;color:#fff;margin:.4rem 0 1rem;width:100%}</style></head><body><main><header><div><h1>Networks</h1><nav><a href="/">Dashboard</a><a href="/users">Users</a><a href="/networks">Networks</a></nav></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header><section><h2>Select user</h2><form method="get" action="/networks"><label>soju username<input name="user" value="{{.User}}" required></label><button type="submit">Load networks</button></form></section>{{if .Notice}}<section><p class="ok">{{.Notice}}</p></section>{{end}}{{if .User}}<section><h2>Network status for {{.User}}</h2><p class="muted">Admin socket: <code>{{.SocketPath}}</code></p>{{if .Error}}<p class="bad">{{.Error}}</p>{{else}}<pre>{{range .Lines}}{{.}}
{{else}}No network status lines returned.{{end}}</pre>{{end}}</section><section><h2>Create network</h2><form method="post" action="/networks/create"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><label>Address<input name="addr" placeholder="irc.libera.chat or ircs://irc.libera.chat:6697" required></label><label>Name<input name="network_name"></label><label>Nick<input name="nick"></label><label>IRC username<input name="irc_username"></label><label>Real name<input name="realname"></label><label>Enabled<select name="enabled"><option value="true">Yes</option><option value="false">No</option></select></label><button type="submit">Create network</button></form></section><section><h2>Update network</h2><form method="post" action="/networks/update"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><label>Current network name<input name="name" required></label><label>New address (optional)<input name="addr"></label><label>New name (optional)<input name="network_name"></label><label>Nick (optional)<input name="nick"></label><label>IRC username (optional)<input name="irc_username"></label><label>Real name (optional)<input name="realname"></label><label>Enabled<select name="enabled"><option value="">Unchanged</option><option value="true">Yes</option><option value="false">No</option></select></label><button type="submit">Update network</button></form></section><section><h2>Delete network</h2><form method="post" action="/networks/delete"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><label>Network name<input name="name" required></label><label>Type <code>delete</code> to confirm<input name="confirm" required></label><button type="submit">Delete network</button></form></section>{{end}}</main></body></html>`
