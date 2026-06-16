package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/crolbar/brok/brokd/dbus"
	"github.com/crolbar/brok/share"
)

const (
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

const (
	BROKCTL_UPDATE_NEXT         = "next"
	BROKCTL_UPDATE_PREV         = "prev"
	BROKCTL_UPDATE_PLAY_PAUSE   = "play-pause"
	BROKCTL_UPDATE_FOCUS_CHANGE = "focus-changed"
)

type M struct {
	dbusConn *dbus.Dbus

	listener       net.Listener
	listeningConns []*net.Conn

	// key == player.id
	players map[string]*Player
	// playersOrder[0] is the focused player, 1, 2 are in order below it
	playersOrder []string
	// used for caching id maps
	playersIDsMap map[string]string

	// indicates that we have recived a msg from brokctl, also what msg it was
	// will be send in the next write to the listeners
	brokctlUpdate string

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
			m.brokctlUpdate = BROKCTL_UPDATE_FOCUS_CHANGE
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
	msg := m.dbusConn.Call(
		"org.freedesktop.DBus.AddMatch",
		dbus.WithBody([]byte("type='signal',interface='org.freedesktop.DBus.Properties'")),
	)
	if msg.Type == dbus.MSG_ERROR {
		panic(msg.String())
	}

	msg = m.dbusConn.Call(
		"org.freedesktop.DBus.AddMatch",
		dbus.WithBody([]byte("type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'")),
	)
	if msg.Type == dbus.MSG_ERROR {
		panic(msg.String())
	}

	var sig_ch chan dbus.Msg = make(chan dbus.Msg, 1000)
	m.dbusConn.SetSignalCh(&sig_ch)
	for !m.quit {
		sig := <-sig_ch

		if sig.Type != dbus.MSG_SIGNAL {
			panic("non signal in sig chan")
		}

		if sig.Headers[dbus.FieldPath].Value.(string) == "/org/freedesktop/DBus" &&
			sig.Headers[dbus.FieldMember].Value.(string) == "NameOwnerChanged" {
			m.handleNameOwnerChanged(sig.Body)
			continue
		}

		if sig.Headers[dbus.FieldMember].Value.(string) != "PropertiesChanged" ||
			sig.Headers[dbus.FieldPath].Value.(string) != "/org/mpris/MediaPlayer2" {
			continue
		}

		sender := sig.Headers[dbus.FieldSender].Value.(string)
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

		if sender == "org.mpris.MediaPlayer2.playerctld" {
			continue
		}

		// on any action from this player, focus it
		m.focusPlayer(sender)

		props := dbus.ParsePropertiesChanged(sig.Body)
		if up := m.upPlayerProps(sender, props); up {
			// fmt.Printf("\x1b[34m[%s]\x1b[m %s\n", sender, props)
			m.writeToListeners()
			// m.printPlayers()
		}
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

	conn, err := dbus.NewSession()
	if err != nil {
		panic(err)
	}
	defer conn.C.Close()
	fmt.Printf("\x1b[34mConnected to dbus (%s)\x1b[m\n", conn.Name)

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
