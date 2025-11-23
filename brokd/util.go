package main

import (
	"encoding/binary"
	"strings"
)

func getMetadataVal(key string, metadata string) string {
	var (
		valStartIdx = strings.Index(metadata, key) + len(key)
		valStart    = ""
		val         = ""
	)

	if valStartIdx != len(key)-1 {
		valStart = metadata[valStartIdx:]
		val = metadata[valStartIdx : valStartIdx+strings.Index(valStart, ">")]
	}

	if key == artistKey {
		return val[1 : len(val)-1]
	}

	return val
}

func getUint16Bytes(i uint16) []byte {
	size := make([]byte, 2)
	binary.LittleEndian.PutUint16(size, i)
	return size
}
