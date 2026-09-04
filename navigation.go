package main

import "net/url"

type userRow struct {
	userStatus
	NetworksURL string
}

type networkRow struct {
	networkStatus
	ChannelsURL string
	SecurityURL string
}

func makeUserRows(users []userStatus) []userRow {
	rows := make([]userRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, userRow{
			userStatus:  user,
			NetworksURL: pageURL("/networks", url.Values{"user": {user.Username}}),
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
