package main

import (
	"slices"
	"strings"
)

func (m *M) handleNameOwnerChanged(player string) {
	if !strings.HasPrefix(player, "org.mpris.MediaPlayer2") {
		return
	}

	if v, ok := m.playersIDsMap[player]; ok {
		delete(m.playersIDsMap, player)
		delete(m.playersIDsMap, v)

		// fmt.Printf("\x1b[34mdel player: %s\x1b[m\n", player)
	} else {
		// fmt.Printf("\x1b[34mnew player: %s\x1b[m\n", player)
	}

	m.upPlayers()
}

func (m *M) getPlayerName(sender string) string {
	for _, player := range m.playersOrder {
		var owner string
		err := m.dbusConn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, player).Store(&owner)
		if err != nil {
			continue
		}

		if owner == sender {
			return player
		}
	}

	return ""
}

func (m *M) upPlayers() {
	var names []string
	err := m.dbusConn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		panic("Failed to get list of owned names: " + err.Error())
	}

	// fmt.Println("Currently owned names on the session bus:")
	// for _, v := range names {
	// 	fmt.Println(v)
	// }

	// Find MPRIS players
	var players []string
	for _, name := range names {
		if !strings.HasPrefix(name, "org.mpris.MediaPlayer2") {
			continue
		}
		if name == "org.mpris.MediaPlayer2.playerctld" {
			continue
		}

		players = append(players, name)
	}

	if len(players) == 0 {
		m.players = nil
		m.playersOrder = nil
		return
	}

	// initializer
	if m.players == nil || m.playersOrder == nil {
		m.playersOrder = players
		m.players = make(map[string]*Player, len(players))
		for _, v := range players {
			m.players[v] = &Player{
				id: v,

				name:   "",
				status: Paused,
				artUri: "",
				title:  "",
				artist: "",
			}
		}

		return
	}

	// find&add missing players
	for _, pID := range players {
		if _, ok := m.players[pID]; ok {
			continue
		}

		// missing player, add it
		m.playersOrder = append(m.playersOrder, pID)
		m.players[pID] = &Player{
			id: pID,

			name:   "",
			status: Paused,
			artUri: "",
			title:  "",
			artist: "",
		}
	}

	// find&remove redundant players
	for pID := range m.players {
		if slices.Contains(players, pID) {
			continue
		}

		// redundant
		delete(m.players, pID)
		m.deletePlayerInOrder(pID)
	}
}
