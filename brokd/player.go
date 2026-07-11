package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/crolbar/brok/brokd/dbus"
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
			return
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

func marshalString(s string) string {
	out, err := json.Marshal(s)
	if err != nil {
		fmt.Println("\x1b[31mError while marshling title\x1b[m")
		return s
	}
	return string(out)
}

func (m *M) getPlayersJson() string {
	var sb strings.Builder

	sb.WriteByte('{')

	sb.WriteString(`"brokctl-update":"` + m.brokctlUpdate + `",`)

	if len(m.brokctlUpdate) > 0 {
		m.brokctlUpdate = ""
	}

	sb.WriteString("\"players\":[")
	for i, pID := range m.playersOrder {
		p := m.players[pID]

		sb.WriteByte('{')

		sb.WriteString(`"id":"` + p.id + `",`)
		sb.WriteString(`"name":"` + p.name + `",`)

		sb.WriteString(`"status":`)
		sb.WriteRune('0' + rune(p.status))
		sb.WriteByte(',')

		sb.WriteString(`"title":` + marshalString(p.title) + `,`)
		sb.WriteString(`"artist":` + marshalString(p.artist) + `,`)
		sb.WriteString(`"artUri":` + marshalString(p.artUri))

		sb.WriteByte('}')

		if i != len(m.playersOrder)-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
	sb.WriteByte('}')

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
	if len(props) == 0 {
		fmt.Println("empty")
		return false
	}
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

		switch event {
		case "PlaybackStatus":
			{
				value := strings.ReplaceAll(v.Value.(string), "\"", "")
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

		case "Metadata":
			{
				value := v.Value.(map[string]dbus.Variant)

				if vari, ok := value["mpris:artUrl"]; ok {
					upIfNE(&player.artUri, vari.Value.(string), &haveUpdate)
				}
				if vari, ok := value["xesam:title"]; ok {
					upIfNE(&player.title, vari.Value.(string), &haveUpdate)
				}
				if vari, ok := value["xesam:artist"]; ok {
					upIfNE(&player.artist, vari.Value.([]string)[0], &haveUpdate)
				}
			}
		}
	}

	return haveUpdate
}

func (m *M) writeToListeners() {
	if len(m.listeningConns) != 0 {
		json := m.getPlayersJson()
		for _, conn := range m.listeningConns {
			size := make([]byte, 2)
			binary.LittleEndian.PutUint16(size, uint16(len(json)))
			(*conn).Write(append(size, []byte(json)...))
		}
	}
}

func (m *M) writePlayersTo(conn *net.Conn) {
	json := m.getPlayersJson()
	size := make([]byte, 2)
	binary.LittleEndian.PutUint16(size, uint16(len(json)))
	(*conn).Write(append(size, []byte(json)...))
}

func (m *M) next(pIDX int) {
	pID := m.playersOrder[pIDX]

	msg := m.dbusConn.Call("org.mpris.MediaPlayer2.Player.Next",
		dbus.WithDest(pID),
		dbus.WithPath("/org/mpris/MediaPlayer2"),
	)
	if msg.Type == dbus.MSG_ERROR {
		panic(msg.String())
	}

	if pIDX != 0 {
		m.focusPlayer(pID)
	}
	m.brokctlUpdate = BROKCTL_UPDATE_NEXT
}

func (m *M) prev(pIDX int) {
	pID := m.playersOrder[pIDX]

	msg := m.dbusConn.Call("org.mpris.MediaPlayer2.Player.Previous",
		dbus.WithDest(pID),
		dbus.WithPath("/org/mpris/MediaPlayer2"),
	)
	if msg.Type == dbus.MSG_ERROR {
		panic(msg.String())
	}

	if pIDX != 0 {
		m.focusPlayer(pID)
	}
	m.brokctlUpdate = BROKCTL_UPDATE_PREV
}

func (m *M) playPause(pIDX int) {
	pID := m.playersOrder[pIDX]

	msg := m.dbusConn.Call("org.mpris.MediaPlayer2.Player.PlayPause",
		dbus.WithDest(pID),
		dbus.WithPath("/org/mpris/MediaPlayer2"),
	)
	if msg.Type == dbus.MSG_ERROR {
		panic(msg.String())
	}

	if pIDX != 0 {
		m.focusPlayer(pID)
	}
	m.brokctlUpdate = BROKCTL_UPDATE_PLAY_PAUSE
}
