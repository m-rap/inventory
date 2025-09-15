package main

import (
	"fmt"
	"inventory"
	"inventoryrpc"
	"os"
)

func main() {
	// Example encode/decode roundtrip
	enc := inventoryrpc.NewEncoder()
	it := inventory.Item{ID: "I1", Name: "Steel", Description: "Raw material", Unit: "kg"}
	enc.WriteItem(it)
	rawMsg := inventoryrpc.EncodeMessage(inventoryrpc.TypeItem, enc.Buf.Bytes())

	msgBuffer := inventoryrpc.MessageBuffer{}
	receivedMsgs := msgBuffer.Feed(rawMsg)

	for i := range receivedMsgs {
		msgType, dec, err := inventoryrpc.DecodeMessage(receivedMsgs[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding message %d: %v\n", i, err)
			continue
		}
		switch msgType {
		case inventoryrpc.TypeItem:
			it2, err := dec.ReadItem()
			if err != nil {
				panic(err)
			}
			fmt.Printf("Decoded Item: %+v\n", it2)
		default:
			fmt.Printf("Unknown message type: %d\n", msgType)
		}
	}

	// // Note: UDP demo requires network setup

	// udpreceiver, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 2001})
	// if err != nil {
	// 	panic("can't create udp")
	// }
	// defer udpreceiver.Close()

	// udpsender, err := net.ListenUDP("udp", nil)
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	// defer udpsender.Close()
}
