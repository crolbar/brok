package main

import (
	"bytes"
	"encoding/binary"
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

	if sig, ok := m.headers[FieldSignature]; ok && len(sig.value.([]uint8)) > 0 {
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

func ParseStringArray(data []byte) []string {
	var a []string = make([]string, 0)

	i := 4
	for i < len(data) {
		var strLen uint32
		binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &strLen)
		i += 4

		str := string(data[i : i+int(strLen)])
		i += int(strLen)

		a = append(a, str)

		// null term
		i += 1

		// padding
		if i%4 != 0 {
			i += 4 - (i % 4)
		}
	}

	return a
}

func ParseArrayStringVariant(data []byte) map[string]Variant {
	m := make(map[string]Variant)

	i := 0
	for i < len(data) {
		// fmt.Println("loop start", data[i:])
		var propLen uint32
		binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &propLen)
		i += 4

		prop := string(data[i : i+int(propLen)])
		i += int(propLen) + 1

		// fmt.Println("prop", prop)

		sigLen := data[i]
		i += 1
		sig := string(data[i : i+int(sigLen)])
		i += int(sigLen) + 1

		var value any
		switch sig {
		case "s", "o":
			if i%4 != 0 {
				i += 4 - (i % 4)
			}
			var strLen uint32
			binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &strLen)
			i += 4

			str := string(data[i : i+int(strLen)])
			i += int(strLen) + 1

			value = str

		case "a{sv}":
			if i%4 != 0 {
				i += 4 - (i % 4)
			}
			var arraySize uint32
			binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &arraySize)
			i += 4

			M := ParseArrayStringVariant(data[i : i+int(arraySize)])
			value = M

		case "x", "t":
			if i%8 != 0 {
				i += 8 - (i % 8)
			}
			var n int64
			binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &n)
			i += 8
			value = n

		case "d":
			if i%8 != 0 {
				i += 8 - (i % 8)
			}
			var n float64
			binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &n)
			i += 8
			value = n

		case "i":
			if i%4 != 0 {
				i += 4 - (i % 4)
			}
			var n int32
			binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &n)
			i += 4
			value = n

		case "as":
			if i%4 != 0 {
				i += 4 - (i % 4)
			}
			var arrLen uint32
			binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &arrLen)
			a := ParseStringArray(data[i : i+int(arrLen)])
			i += 4
			i += int(arrLen)

			value = a

		case "b":
			if i%4 != 0 {
				i += 4 - (i % 4)
			}

			if data[i] == 0 {
				value = false
			} else {
				value = true
			}
			i += 4

		default:
			panic("unknown sig: " + sig)
		}
		// fmt.Println("value", value)

		// end of dict entry, dict entry has alignment of 8
		if i%8 != 0 {
			i += 8 - (i % 8)
		}

		// fmt.Println()
		// fmt.Println()

		m[prop] = Variant{sig: Signature{sig}, value: value}
	}

	return m
}

func ParsePropertiesChanged(data []byte) map[string]Variant {
	i := 0
	var interNameLen uint32
	binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &interNameLen)
	i += 4

	interName := string(data[i : i+int(interNameLen)])
	i += int(interNameLen)
	if interName != "org.mpris.MediaPlayer2.Player" {
		panic("signal PropertiesChanged not comming from mpris player?")
	}

	if i%4 != 0 {
		i += 4 - (i % 4)
	}

	var changedPropertiesSize uint32
	binary.Read(bytes.NewBuffer(data[i:]), binary.LittleEndian, &changedPropertiesSize)
	i += 4

	changedProperties := ParseArrayStringVariant(data[i : i+int(changedPropertiesSize)])
	return changedProperties
}
