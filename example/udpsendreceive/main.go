package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"inventory"
	"inventoryrpc"
	"log"
	"net"
)

// ---------------- UDP Transport ----------------

func SendUDP(conn *net.UDPConn, addr *net.UDPAddr, data []byte) error {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)
	_, err := conn.WriteToUDP(buf.Bytes(), addr)
	return err
}

type UDPBuffer struct {
	buf bytes.Buffer
}

func (u *UDPBuffer) Feed(packet []byte) ([][]byte, error) {
	u.buf.Write(packet)
	var messages [][]byte

	for {
		if u.buf.Len() < 4 {
			break
		}
		length := binary.LittleEndian.Uint32(u.buf.Bytes()[:4])
		if u.buf.Len() < int(4+length) {
			break
		}
		full := make([]byte, length)
		copy(full, u.buf.Bytes()[4:4+length])
		u.buf.Next(4 + int(length))
		messages = append(messages, full)
	}
	return messages, nil
}

// ---------------- Demo ----------------

func main() {
	// Example encode/decode roundtrip
	enc := inventoryrpc.NewEncoder()
	it := inventory.Item{ID: "I1", Name: "Steel", Description: "Raw material", Unit: "kg"}
	enc.WriteItem(it)

	dec := inventoryrpc.NewDecoder(enc.Bytes())
	it2, err := dec.ReadItem()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Roundtrip Item: %+v\n", it2)

	// Note: UDP demo requires network setup

	udpreceiver, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 2001})
	if err != nil {
		panic("can't create udp")
	}
	defer udpreceiver.Close()

	udpsender, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Fatal(err.Error())
	}

	defer udpsender.Close()
}
