package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type channelsPageData struct {
	CSRF        string
	User        string
	Network     string
	NetworksURL string
	SecurityURL string
	Lines       []string
	Channels    []channelStatus
	Error       string
	Notice      string
}

func (a *app) channelsPage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("user"))
	network := strings.TrimSpace(r.URL.Query().Get("network"))
	data := channelsPageData{CSRF: a.csrfToken(), User: username, Network: network, Notice: channelNoticeText(r.URL.Query().Get("ok"))}
	if username != "" {
		data.NetworksURL = pageURL("/networks", url.Values{"user": {username}})
	}
	if username != "" && network != "" {
		data.SecurityURL = pageURL("/security", url.Values{"user": {username}, "network": {network}})
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		lines, err := a.adminClient().Run(ctx, "user", "run", username, "channel", "status", "-network", network)
		data.Lines = lines
		data.Channels = parseChannelStatuses(lines)
		if err != nil {
			data.Error = err.Error()
		}
	}
	render(w, channelsTemplate, data)
}

func (a *app) createChannel(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("user"))
	network := strings.TrimSpace(r.FormValue("network"))
	channel := strings.TrimSpace(r.FormValue("channel"))
	if !validChannelTarget(username, network, channel) {
		http.Error(w, "user, network and valid channel name are required", http.StatusBadRequest)
		return
	}
	words := []string{"user", "run", username, "channel", "create", channel + "/" + network}
	words = appendChannelOptions(words, r)
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, words...); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	redirectChannels(w, r, username, network, "created")
}

func (a *app) updateChannel(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("user"))
	network := strings.TrimSpace(r.FormValue("network"))
	channel := strings.TrimSpace(r.FormValue("channel"))
	if !validChannelTarget(username, network, channel) {
		http.Error(w, "user, network and valid channel name are required", http.StatusBadRequest)
		return
	}
	words := []string{"user", "run", username, "channel", "update", channel + "/" + network}
	words = appendChannelOptions(words, r)
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, words...); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	redirectChannels(w, r, username, network, "updated")
}

func (a *app) deleteChannel(w http.ResponseWriter, r *http.Request) {
	if !a.prepareAdminPost(w, r) {
		return
	}
	username := strings.TrimSpace(r.FormValue("user"))
	network := strings.TrimSpace(r.FormValue("network"))
	channel := strings.TrimSpace(r.FormValue("channel"))
	if !validChannelTarget(username, network, channel) || r.FormValue("confirm") != "delete" {
		http.Error(w, "valid target and delete confirmation are required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	if _, err := a.adminClient().Run(ctx, "user", "run", username, "channel", "delete", channel+"/"+network); err != nil {
		http.Error(w, "soju: "+err.Error(), http.StatusBadGateway)
		return
	}
	redirectChannels(w, r, username, network, "deleted")
}

func appendChannelOptions(words []string, r *http.Request) []string {
	for _, field := range []struct{ form, flag string }{
		{"detached", "-detached"},
		{"relay_detached", "-relay-detached"},
		{"reattach_on", "-reattach-on"},
		{"detach_after", "-detach-after"},
		{"detach_on", "-detach-on"},
	} {
		if v := strings.TrimSpace(r.FormValue(field.form)); v != "" {
			words = append(words, field.flag, v)
		}
	}
	return words
}

func validChannelTarget(username, network, channel string) bool {
	if username == "" || network == "" || channel == "" {
		return false
	}
	if strings.ContainsAny(username+network+channel, "\r\n\x00") {
		return false
	}
	if strings.Contains(network, "/") || strings.ContainsAny(network, " \t") {
		return false
	}
	return (strings.HasPrefix(channel, "#") || strings.HasPrefix(channel, "&")) && !strings.ContainsAny(channel, " ,/\t")
}

func redirectChannels(w http.ResponseWriter, r *http.Request, username, network, ok string) {
	q := url.Values{"user": {username}, "network": {network}, "ok": {ok}}
	http.Redirect(w, r, "/channels?"+q.Encode(), http.StatusSeeOther)
}

func channelNoticeText(v string) string {
	switch v {
	case "created":
		return "Channel saved and joined when the network is connected."
	case "updated":
		return "Channel settings updated."
	case "deleted":
		return "Channel deleted."
	default:
		return ""
	}
}

const channelsTemplate = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Channels · soju-web</title><style>` + baseCSS + ` nav a{color:#fbbf24;margin-right:1rem}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:.7rem;border-bottom:1px solid #374151}.badge{display:inline-block;padding:.15rem .5rem;border-radius:999px;background:#374151}.badge.ok{background:#064e3b}.badge.bad{background:#7f1d1d}.badge.warn{background:#78350f}.context a{color:#fbbf24;margin-right:.75rem}select{font:inherit;padding:.7rem;border-radius:8px;border:1px solid #4b5563;background:#111827;color:#fff;margin:.4rem 0 1rem;width:100%}</style></head><body><main><header><div><h1>Channels</h1><nav><a href="/">Dashboard</a><a href="/users">Users</a><a href="/networks">Networks</a><a href="/channels">Channels</a><a href="/security">Security</a></nav></div><form method="post" action="/logout"><button type="submit">Sign out</button></form></header><section><h2>Select target</h2><form method="get" action="/channels"><label>soju username<input name="user" value="{{.User}}" required></label><label>Network name<input name="network" value="{{.Network}}" required></label><button type="submit">Load channels</button></form></section>{{if .Notice}}<section><p class="ok">{{.Notice}}</p></section>{{end}}{{if and .User .Network}}<section><h2>Channel status</h2><p class="muted context"><a href="{{.NetworksURL}}">← Networks</a><a href="{{.SecurityURL}}">Security →</a> User <code>{{.User}}</code>, network <code>{{.Network}}</code></p>{{if .Error}}<p class="bad">{{.Error}}</p>{{else if .Channels}}<table><thead><tr><th>Channel</th><th>Status</th><th>Attachment</th></tr></thead><tbody>{{range .Channels}}<tr><td><code>{{.Name}}</code></td><td>{{if eq .State "joined"}}<span class="badge ok">joined</span>{{else if eq .State "parted"}}<span class="badge warn">parted</span>{{else}}<span class="badge bad">{{.State}}</span>{{end}}</td><td>{{if .Detached}}<span class="badge warn">detached</span>{{else}}attached{{end}}</td></tr>{{end}}</tbody></table>{{else}}<p class="muted">No saved channels.</p>{{end}}</section><section><h2>Create channel</h2><form method="post" action="/channels/create"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><input type="hidden" name="network" value="{{.Network}}"><label>Channel<input name="channel" placeholder="#soju" required></label><label>Detached<select name="detached"><option value="">Default</option><option value="false">No</option><option value="true">Yes</option></select></label><label>Relay while detached<select name="relay_detached"><option value="">Default</option><option value="default">default</option><option value="none">none</option><option value="highlight">highlight</option><option value="message">message</option></select></label><label>Reattach on<select name="reattach_on"><option value="">Default</option><option value="default">default</option><option value="none">none</option><option value="highlight">highlight</option><option value="message">message</option></select></label><label>Detach after<input name="detach_after" placeholder="0, 300s, 22h30m"></label><label>Detach on<select name="detach_on"><option value="">Default</option><option value="default">default</option><option value="none">none</option><option value="highlight">highlight</option><option value="message">message</option></select></label><button type="submit">Create channel</button></form></section><section><h2>Update channel</h2><form method="post" action="/channels/update"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><input type="hidden" name="network" value="{{.Network}}"><label>Channel<input name="channel" required></label><label>Detached<select name="detached"><option value="">Unchanged</option><option value="false">No</option><option value="true">Yes</option></select></label><label>Relay while detached<select name="relay_detached"><option value="">Unchanged</option><option value="default">default</option><option value="none">none</option><option value="highlight">highlight</option><option value="message">message</option></select></label><label>Reattach on<select name="reattach_on"><option value="">Unchanged</option><option value="default">default</option><option value="none">none</option><option value="highlight">highlight</option><option value="message">message</option></select></label><label>Detach after<input name="detach_after" placeholder="leave empty to keep current"></label><label>Detach on<select name="detach_on"><option value="">Unchanged</option><option value="default">default</option><option value="none">none</option><option value="highlight">highlight</option><option value="message">message</option></select></label><button type="submit">Update channel</button></form></section><section><h2>Delete channel</h2><form method="post" action="/channels/delete"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="user" value="{{.User}}"><input type="hidden" name="network" value="{{.Network}}"><label>Channel<input name="channel" required></label><label>Type <code>delete</code> to confirm<input name="confirm" required></label><button type="submit">Delete channel</button></form></section>{{end}}</main></body></html>`
