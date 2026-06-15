package main

import (
	"bytes"
	"fmt"
	"strings"
)

type CallOpt func(*Call)

func WithPath(path string) func(*Call) {
	return func(c *Call) {
		c.path = path
	}
}
func WithDest(dest string) func(*Call) {
	return func(c *Call) {
		c.dest = dest
	}
}
func WithBody(body []byte) func(*Call) {
	return func(c *Call) {
		c.body = body
	}
}
func WithFlags(flags byte) func(*Call) {
	return func(c *Call) {
		c.flags = flags
	}
}

func (d *Dbus) getSerial() int {
	d.serial += 1
	return d.serial
}

func (m Msg) String() string {
	var sb strings.Builder

	if m.headers == nil {
		return "empty msg"
	}

	sb.WriteString(fmt.Sprintf("Msg: %d\n", m.Type))
	sb.WriteString("  Headers:\n")
	for k, v := range m.headers {
		sb.WriteString("    [")
		switch k {
		case FieldPath:
			sb.WriteString("FieldPath")
		case FieldInterface:
			sb.WriteString("FieldInterface")
		case FieldMember:
			sb.WriteString("FieldMember")
		case FieldErrorName:
			sb.WriteString("FieldErrorName")
		case FieldReplySerial:
			sb.WriteString("FieldReplySerial")
		case FieldDestination:
			sb.WriteString("FieldDestination")
		case FieldSender:
			sb.WriteString("FieldSender")
		case FieldSignature:
			sb.WriteString("FieldSignature")
		case FieldUnixFDs:
			sb.WriteString("FieldUnixFDs")
		}
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprintf("%q", v.value))
		// sb.WriteString(fmt.Sprintf("%s", v.sig))
		sb.WriteString("]\n")
	}

	if sig, ok := m.headers[FieldSignature]; ok && len(sig.value.([]uint8)) > 0{
		switch string(m.headers[FieldSignature].value.([]uint8)[0]) {
		case "s":
			sb.WriteString(fmt.Sprintf("  Body: %s", string(m.body)))
		case "b":
			if m.body[0] == 0 {
				sb.WriteString("  Body: false")
			} else {
				sb.WriteString("  Body: true")
			}
		default:
			sb.WriteString(fmt.Sprintf("  Body: %v", m.body))
		}
	} else {
		sb.WriteString(fmt.Sprintf("  Body: %v", m.body))
	}

	return sb.String()
}

func pad(b *bytes.Buffer, align int) {
	for b.Len()%align != 0 {
		b.WriteByte(0)
	}
}
