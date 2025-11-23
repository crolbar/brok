package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/crolbar/brok/share"
	"github.com/godbus/dbus"
)

const (
	artUriKey = "mpris:artUrl: <"
	artistKey = "xesam:artist: <"
	titleKey  = "xesam:title: <"

	idPrefix    = "org.mpris.MediaPlayer2."
	idPrefixLen = len("org.mpris.MediaPlayer2.")
)

type Status = int

const (
	Playing Status = iota
	Paused
)

type Player struct {
	id     string
	name   string // id without the org.mpris.MediaPlayer2. prefix
	status Status
	artUri string
	title  string
	artist string
}

type M struct {
	dbusConn *dbus.Conn

	listeningConns []*net.Conn

	// key == player.id
	players map[string]*Player
	// playersOrder[0] is the focused player, 1, 2 are in order below it
	playersOrder []string

	// used for caching id maps
	playersIDsMap map[string]string

	listener net.Listener

	quit bool
}

func (m *M) handleMsg(msg string, conn *net.Conn) {
	switch msg {
	case share.MSG_NEXT:
		m.next(0)
	case share.MSG_PREV:
		m.prev(0)
	case share.MSG_PLAY_PAUSE:
		m.playPause(0)

	case share.MSG_SUB:
		m.listeningConns = append(m.listeningConns, conn)

	case "quit":
		m.quit = true
		m.listener.Close()
	}

	if len(msg) > share.MSG_FOCUS_LEN && msg[:share.MSG_FOCUS_LEN] == share.MSG_FOCUS {
		pID := msg[share.MSG_FOCUS_LEN+1:]
		if _, ok := m.players[pID]; ok {
			m.focusPlayer(pID)

			(*conn).Write([]byte("ok"))
		} else {
			(*conn).Write([]byte("incorrect id, id is not in players"))
		}

		m.writeToListeners()
	}
}

func (m *M) handleConn(conn net.Conn) {
	defer func() {
		conn.Close()
		for i, c := range m.listeningConns {
			if c != &conn {
				continue
			}
			m.listeningConns = append(m.listeningConns[:i], m.listeningConns[i+1:]...)
		}
	}()
	for {
		buf := make([]byte, share.MAX_MSG_LEN)
		n, err := conn.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				return
			}

			panic(err)
		}

		data := buf[:n]

		if strings.HasPrefix(string(data), "msg:") {
			m.handleMsg(string(data[4:]), &conn)
		}
	}
}

func (m *M) dbusListener() {
	call := m.dbusConn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='org.freedesktop.DBus.Properties'",
	)
	if call.Err != nil {
		panic(call.Err)
	}

	call = m.dbusConn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'",
	)
	if call.Err != nil {
		panic(call.Err)
	}

	var sig_ch chan *dbus.Signal = make(chan *dbus.Signal)
	m.dbusConn.Signal(sig_ch)
	for !m.quit {
		sig := <-sig_ch

		if sig.Name == "org.freedesktop.DBus.NameOwnerChanged" {
			m.handleNameOwnerChanged(sig.Body[0].(string))
			continue
		}

		sender := sig.Sender
		if !strings.HasPrefix(sender, "org.mpris.MediaPlayer2") {
			var p string

			p = m.playersIDsMap[sender]
			if len(p) == 0 {
				p = m.getPlayerName(sender)
				if len(p) == 0 {
					continue
				}

				m.playersIDsMap[p] = sender
				m.playersIDsMap[sender] = p
				sender = p
			} else {
				sender = p
			}
		}

		// fmt.Printf("\x1b[34m[%s]\x1b[m %s\n", sender, sig.Body)

		if sender == "org.mpris.MediaPlayer2.playerctld" {
			continue
		}

		// on any action from this player, focus it
		m.focusPlayer(sender)

		m.upPlayerProps(sender, sig.Body[1].(map[string]dbus.Variant))

		m.writeToListeners()

		/*
			BODY:
			[org.mpris.MediaPlayer2.Player map[PlaybackStatus:"Playing"] []]
			[org.mpris.MediaPlayer2.Player map[PlaybackStatus:"Paused"] []]
			[org.mpris.MediaPlayer2.Player map[Metadata:{"mpris:artUrl": <"...

			[org.mpris.MediaPlayer2.firefox.instance_1_26] [org.mpris.MediaPlayer2.Player map[CanGoNext:false] []]
			[org.mpris.MediaPlayer2.firefox.instance_1_26] [org.mpris.MediaPlayer2.Player map[CanGoPrevious:false] []]
		*/
	}
}

func main() {
	if _, err := os.Stat(share.SockPath); err == nil {
		fmt.Println("\x1b[33mremoving old socket\x1b[m")
		err = os.Remove(share.SockPath)

		if err != nil {
			panic(err)
		}
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("\x1b[34mConnected to dbus\x1b[m")

	listener, err := net.Listen("unix", share.SockPath)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Printf("\x1b[34mConnected to unix socket at %s\x1b[m\n", share.SockPath)

	m := M{
		quit:           false,
		listener:       listener,
		dbusConn:       conn,
		playersIDsMap:  make(map[string]string),
		listeningConns: make([]*net.Conn, 0),

		playersOrder: nil,
		players:      nil,
	}

	m.upPlayers()

	go m.dbusListener()

	for !m.quit {
		conn, err := m.listener.Accept()
		if err != nil && !m.quit {
			panic(err)
		}

		fmt.Printf("\x1b[34mAccepted conn from addr: %s\x1b[m\n", conn.LocalAddr().String())

		go m.handleConn(conn)
	}
}
