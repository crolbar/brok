package main

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus"
)

func (m *M) focusPlayer(id string) {
	if _, ok := m.players[id]; !ok {
		println("id of focus not in players")
		return
	}

	// remove old id place
	m.deletePlayerInOrder(id)

	// insert id at top
	m.playersOrder = append([]string{id}, m.playersOrder...)
}

func (m *M) deletePlayerInOrder(id string) {
	for i, _id := range m.playersOrder {
		if _id == id {
			m.playersOrder = append(m.playersOrder[:i], m.playersOrder[i+1:]...)
		}
	}
}

func (m *M) printPlayers() {
	fmt.Println("players: ================")
	for _, id := range m.playersOrder {
		fmt.Println("player ", id)
		p := m.players[id]
		fmt.Println("   name: ", p.name)
		fmt.Println("   status: ", p.status)
		fmt.Println("   artUri: ", p.artUri)
		fmt.Println("   artist: ", p.artist)
		fmt.Println("   title: ", p.title)
	}
	fmt.Println()
	fmt.Println()
}

func (m *M) getPlayersJson() string {
	var sb strings.Builder

	sb.WriteByte('[')
	for i, pID := range m.playersOrder {
		p := m.players[pID]

		sb.WriteString(fmt.Sprintf(
			"{"+
				`"id":"%s",`+
				`"name":"%s",`+
				`"status":%d,`+
				`"title":"%s",`+
				`"artist":"%s",`+
				`"artUri":"%s"`+
				"}",
			p.id,
			p.name,
			p.status,
			p.title,
			p.artist,
			p.artUri,
		))

		if i != len(m.playersOrder)-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')

	return sb.String()
}

func upIfNE[T string | int](curr *T, new T, up *bool) {
	if *curr == new {
		return
	}

	*curr = new
	*up = true
}

func (m *M) upPlayerProps(pID string, props map[string]dbus.Variant) bool {
	player := m.players[pID]

	if len(player.name) == 0 {
		pre := pID[idPrefixLen:]
		sufIdx := strings.Index(pre, ".")
		if sufIdx != -1 {
			player.name = pre[:sufIdx]
		} else {
			player.name = pre
		}
	}

	haveUpdate := false
	for k, v := range props {
		event := strings.ReplaceAll(k, "\"", "")
		value := strings.ReplaceAll(v.String(), "\"", "")

		if event == "PlaybackStatus" {
			switch value {
			case "Playing":
				// player.status = Playing
				upIfNE(&player.status, Playing, &haveUpdate)
			case "Stopped":
				fallthrough
			case "Paused":
				upIfNE(&player.status, Paused, &haveUpdate)
			}
		}

		if event == "Metadata" {
			upIfNE(&player.artUri, getMetadataVal(artUriKey, value), &haveUpdate)
			upIfNE(&player.title, getMetadataVal(titleKey, value), &haveUpdate)
			upIfNE(&player.artist, getMetadataVal(artistKey, value), &haveUpdate)
		}
	}

	return haveUpdate
}

func (m *M) writeToListeners() {
	if len(m.listeningConns) != 0 {
		json := m.getPlayersJson()
		for _, conn := range m.listeningConns {
			(*conn).Write(append(getUint16Bytes(uint16(len(json))), []byte(json)...))
		}
	}
}

func (m *M) next(pIDX int) {
	pID := m.playersOrder[pIDX]

	obj := m.dbusConn.Object(pID, "/org/mpris/MediaPlayer2")
	call := obj.Call("org.mpris.MediaPlayer2.Player.Next", 0)
	if call.Err != nil {
		fmt.Println(call.Err)
	}

	if pIDX != 0 {
		m.focusPlayer(pID)
	}
}

func (m *M) prev(pIDX int) {
	pID := m.playersOrder[pIDX]

	obj := m.dbusConn.Object(pID, "/org/mpris/MediaPlayer2")
	call := obj.Call("org.mpris.MediaPlayer2.Player.Previous", 0)
	if call.Err != nil {
		fmt.Println(call.Err)
	}

	if pIDX != 0 {
		m.focusPlayer(pID)
	}
}

func (m *M) playPause(pIDX int) {
	pID := m.playersOrder[pIDX]

	obj := m.dbusConn.Object(pID, "/org/mpris/MediaPlayer2")
	call := obj.Call("org.mpris.MediaPlayer2.Player.PlayPause", 0)
	if call.Err != nil {
		fmt.Println(call.Err)
	}

	if pIDX != 0 {
		m.focusPlayer(pID)
	}
}
