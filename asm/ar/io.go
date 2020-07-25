package ar

import (
	"encoding/binary"
	"io"
)

func check(err error) {
	if err != nil {
		panic((err))
	}
}

var endian = binary.LittleEndian

func readI8(r io.Reader) (v int8) {
	check(binary.Read(r, endian, &v))
	return
}

func readU8(r io.Reader) (v uint8) {
	check(binary.Read(r, endian, &v))
	return
}

func readI16(r io.Reader) (v int16) {
	check(binary.Read(r, endian, &v))
	return
}

func readU16(r io.Reader) (v uint16) {
	check(binary.Read(r, endian, &v))
	return
}

func readI32(r io.Reader) (v int32) {
	check(binary.Read(r, endian, &v))
	return
}

func readU32(r io.Reader) (v uint32) {
	check(binary.Read(r, endian, &v))
	return
}

func readI64(r io.Reader) (v int64) {
	check(binary.Read(r, endian, &v))
	return
}

func readU64(r io.Reader) (v uint64) {
	check(binary.Read(r, endian, &v))
	return
}

func readf32(r io.Reader) (v float32) {
	check(binary.Read(r, endian, &v))
	return
}

func readf64(r io.Reader) (v float64) {
	check(binary.Read(r, endian, &v))
	return
}

func readBytes(r io.Reader) []byte {
	sz := readU16(r)
	p := make([]byte, sz)
	_, err := io.ReadFull(r, p)
	check(err)
	return p
}

func writeI8(w io.Writer, v int8) {
	check(binary.Write(w, endian, v))
	return
}

func writeU8(w io.Writer, v uint8) {
	check(binary.Write(w, endian, v))
	return
}

func writeI16(w io.Writer, v int16) {
	check(binary.Write(w, endian, v))
	return
}

func writeU16(w io.Writer, v uint16) {
	check(binary.Write(w, endian, v))
	return
}

func writeI32(w io.Writer, v int32) {
	check(binary.Write(w, endian, v))
	return
}

func writeU32(w io.Writer, v uint32) {
	check(binary.Write(w, endian, v))
	return
}

func writeI64(w io.Writer, v int64) {
	check(binary.Write(w, endian, v))
	return
}

func writeU64(w io.Writer, v uint64) {
	check(binary.Write(w, endian, v))
	return
}

func writef32(w io.Writer, v float32) {
	check(binary.Write(w, endian, v))
	return
}

func writef64(w io.Writer, v float64) {
	check(binary.Write(w, endian, v))
	return
}

func writeBytes(w io.Writer, p []byte) {
	writeU16(w, uint16(len(p)))
	_, err := w.Write(p)
	check(err)
}
