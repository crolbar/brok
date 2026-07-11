package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/crolbar/brok/share"
)

type M struct {
	conn net.Conn
}

func (m *M) sendMsg(msg string) {
	if len(msg) > share.MAX_MSG_LEN-4 {
		panic("msg len")
	}

	_, err := m.conn.Write([]byte("msg:" + msg))
	if err != nil {
		panic(err)
	}
}

func (m *M) sendFocusMsg(id string) {
	if len(share.MSG_FOCUS)+len(id) > share.MAX_MSG_LEN-4 {
		panic("msg len")
	}

	_, err := m.conn.Write([]byte("msg:" + share.MSG_FOCUS + ":" + id))
	if err != nil {
		panic(err)
	}

	buf := make([]byte, share.MAX_MSG_LEN)
	n, err := m.conn.Read(buf)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(buf[:n]))
}

func (m *M) listener() {
	for {
		m.readOne()
	}
}

func (m *M) readOne() {
	headerBuf := make([]byte, 2)
	n, err := m.conn.Read(headerBuf)
	if err != nil {
		panic(err)
	}
	if n != 2 {
		panic("no 2 byte header")
	}

	size := binary.LittleEndian.Uint16(headerBuf)

	buf := make([]byte, size)
	n, err = m.conn.Read(buf)
	if err != nil {
		panic(err)
	}

	if n != int(size) {
		panic("incorrect size of body send from server")
	}

	data := buf[:n]

	fmt.Println(string(data))
}

// TODO:
// focus next/prev

func printHelp() {
	fmt.Println("\n" +
		"Usage: brokctl [OPTIONS..]" + "\n\n" +
		"[OPTIONS]" + "\n" +
		"next, --next                   send next call" + "\n" +
		"prev, --prev                   send previous call" + "\n" +
		"play-pause, --play-pause       send play-pause call" + "\n" +
		"focus, --focus [id]            move the player with id to the front" + "\n" +
		"sub, subscribe, --subscribe    listen for changes in mpris, and get players in json" + "\n" +
		"ru, --request-update           get the current players without waiting for update (can be combined with sub i.e. `brokctl ru sub`)")
}

func main() {
	for _, arg := range os.Args {
		if arg == "help" || arg == "--help" || arg == "-h" {
			printHelp()
			return
		}
	}

	conn, err := net.Dial("unix", share.SockPath)
	if err != nil {
		panic(err)
	}

	m := M{conn: conn}

	for i, arg := range os.Args {
		switch arg {
		case "next", "--next":
			m.sendMsg(share.MSG_NEXT)
		case "prev", "--prev", "previous", "--previous":
			m.sendMsg(share.MSG_PREV)
		case "play-pause", "--play-pause":
			m.sendMsg(share.MSG_PLAY_PAUSE)

		case "focus", "--focus":

			if len(os.Args) <= i+1 {
				fmt.Println("no arg provided to focus")
				return
			}

			m.sendFocusMsg(os.Args[i+1])
		case "sub", "subscribe", "--subscribe":
			m.sendMsg(share.MSG_SUB)
			m.listener()

		case "ru", "--request-update":
			m.sendMsg(share.MSG_REQ_UP)
			m.readOne()

		case "quit":
			m.sendMsg("quit")
		default:
			continue
		}
	}

	if len(os.Args) <= 1 {
		printHelp()
		return
	}
}
