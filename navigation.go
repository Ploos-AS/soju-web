package main

import "net/url"

type userRow struct {
	userStatus
	NetworksURL string
	ManageURL   string
	DeleteURL   string
}

type networkRow struct {
	networkStatus
	ChannelsURL string
	SecurityURL string
	ManageURL   string
}

type channelRow struct {
	channelStatus
	ManageURL string
}

func makeUserRows(users []userStatus) []userRow {
	rows := make([]userRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, userRow{
			userStatus:  user,
			NetworksURL: pageURL("/networks", url.Values{"user": {user.Username}}),
			ManageURL:   pageURL("/users", url.Values{"manage": {user.Username}}) + "#manage-user",
			DeleteURL:   pageURL("/users/delete", url.Values{"username": {user.Username}}),
		})
	}
	return rows
}

func makeNetworkRows(username string, networks []networkStatus) []networkRow {
	rows := make([]networkRow, 0, len(networks))
	for _, network := range networks {
		q := url.Values{"user": {username}, "network": {network.Name}}
		rows = append(rows, networkRow{
			networkStatus: network,
			ChannelsURL:   pageURL("/channels", q),
			SecurityURL:   pageURL("/security", q),
			ManageURL:     pageURL("/networks", url.Values{"user": {username}, "manage": {network.Name}}) + "#manage-network",
		})
	}
	return rows
}

func makeChannelRows(username, network string, channels []channelStatus) []channelRow {
	rows := make([]channelRow, 0, len(channels))
	for _, channel := range channels {
		rows = append(rows, channelRow{
			channelStatus: channel,
			ManageURL: pageURL("/channels", url.Values{
				"user":    {username},
				"network": {network},
				"manage":  {channel.Name},
			}) + "#manage-channel",
		})
	}
	return rows
}

func pageURL(path string, values url.Values) string {
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}
