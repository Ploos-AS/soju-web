package main

import (
	"context"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

type deleteUserPageData struct {
	CSRF     string
	Username string
	Token    string
	Error    string
}

func (a *app) deleteUserPage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	data := deleteUserPageData{CSRF: a.csrfToken(), Username: username}
	if username == "" || strings.ContainsAny(username, "\r\n\x00") {
		data.Error = "valid username required"
		render(w, deleteUserTemplate, data)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	lines, err := a.adminClient().Run(ctx, "user", "delete", username)
	if err != nil {
		data.Error = err.Error()
		render(w, deleteUserTemplate, data)
		return
	}
	token, ok := extractUserDeleteToken(lines, username)
	if !ok {
		data.Error = "soju did not return a valid deletion confirmation token"
	} else {
		data.Token = token
	}
	render(w, deleteUserTemplate, data)
}

func (a *app) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	token := strings.TrimSpace(r.FormValue("token"))
	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if username == "" || strings.ContainsAny(username, "\r\n\x00") || !validUserDeleteToken(token) {
		http.Error(w, "valid user deletion target and token are required", http.StatusBadRequest)
		return
	}
	if confirm != "delete "+username {
		http.Error(w, "type the exact deletion confirmation phrase", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, "user", "delete", username, token); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/users?ok=deleted", http.StatusSeeOther)
}

func extractUserDeleteToken(lines []string, username string) (string, bool) {
	const prefix = `To confirm user deletion, send "`
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, `"`) {
			continue
		}
		command := strings.TrimSuffix(strings.TrimPrefix(line, prefix), `"`)
		fields := strings.Fields(command)
		if len(fields) != 4 || fields[0] != "user" || fields[1] != "delete" || fields[2] != username {
			continue
		}
		if validUserDeleteToken(fields[3]) {
			return fields[3], true
		}
	}
	return "", false
}

func validUserDeleteToken(token string) bool {
	if len(token) != 6 {
		return false
	}
	decoded, err := hex.DecodeString(token)
	return err == nil && len(decoded) == 3 && token == strings.ToLower(token)
}

const deleteUserTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Delete user · soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}.danger{border:1px solid #7f1d1d}.bad{color:#fecaca}</style></head><body><main><header><div><h1>Delete user</h1><nav><a href="/">Dashboard</a><a href="/users">Users</a></nav></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header><section class="danger"><h2>Permanent deletion</h2>{{if .Error}}<p class="bad">{{.Error}}</p>{{else}}<p>This will permanently delete soju user <code>{{.Username}}</code> and the user's stored configuration. soju generated the native confirmation token for this exact username.</p><form method="post" action="/users/delete"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="username" value="{{.Username}}"><input type="hidden" name="token" value="{{.Token}}"><label>Type <code>delete {{.Username}}</code> to confirm<input name="confirm" autocomplete="off" required></label><button type="submit">Permanently delete {{.Username}}</button></form>{{end}}<p><a href="/users">← Cancel and return to users</a></p></section></main></body></html>`
