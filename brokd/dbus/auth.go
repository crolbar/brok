package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
)

func (d *Dbus) authReadLine() ([][]byte, error) {
	in := bufio.NewReader(d.C)

	msgBuf, err := in.ReadBytes('\n')
	if err != nil {
		return [][]byte{}, err
	}

	bytes.TrimSuffix(msgBuf, []byte("\r\n"))
	return bytes.Split(msgBuf, []byte{' '}), nil
}

func (d *Dbus) authWriteLine(cmds ...[]byte) error {
	buf := make([]byte, 0)

	for i, c := range cmds {
		buf = append(buf, c...)
		if i != len(cmds)-1 {
			buf = append(buf, ' ')
		}
	}

	buf = append(buf, '\r')
	buf = append(buf, '\n')
	n, err := d.C.Write(buf)
	if n != len(buf) {
		panic("write err n != len(buf)")
	}
	return err
}

func (d Dbus) Auth() error {
	// null byte
	{
		_, _, err := d.C.WriteMsgUnix([]byte{0}, []byte{}, nil)
		if err != nil {
			return err
		}
	}

	// {
	// 	err := d.authWriteLine([]byte("AUTH"))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cmds, err := d.authReadLine()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if string(cmds[0]) != "REJECTED" && string(cmds[1]) != "EXTERNAL" {
	// 		return errors.New("auth protocol error: expected REJECTED AND EXTERNAL")
	// 	}
	// }

	// AUTH EXTERNAL + uid
	{
		uid := strconv.Itoa(os.Getuid())
		b := make([]byte, 2*len(uid))
		hex.Encode(b, []byte(uid))

		d.authWriteLine([]byte("AUTH EXTERNAL"), b)

		cmds, err := d.authReadLine()
		if err != nil {
			return err
		}
		if !bytes.Equal(cmds[0], []byte("OK")) {
			return errors.New("dbus auth: status not OK")
		}
	}

	// err := d.authWriteLine([]byte("NEGOTIATE_UNIX_FD"))
	// if err != nil {
	// 	return err
	// }
	// cmds, err := d.authReadLine()
	// if bytes.Equal(cmds[0], []byte("ERROR")) {
	// 	return errors.New("dbus auth: NEGOTION UNIX FD FAILED")
	// }

	err := d.authWriteLine([]byte("BEGIN"))
	if err != nil {
		return err
	}

	return nil
}
