package main

import (
	"fmt"
	"inventory"
	"inventoryrpc"
	"log"
	"os"

	"github.com/vmihailenco/msgpack/v5"
)

func main() {
	// Example encode/decode roundtrip
	// enc := inventoryrpc.NewEncoder()
	it := inventory.Item{UUID: "I1", Name: "Steel", Description: "Raw material", Unit: "kg"}
	// Marshal to bytes
	itemBin, err := msgpack.Marshal(&it)
	if err != nil {
		log.Fatal(err)
	}

	body := make(map[string][]byte)
	body["function"] = []byte("InsertItem")
	body["item"] = itemBin

	bodyBin, err := msgpack.Marshal(body)

	pktBin, err := inventoryrpc.EncodePacket(0, inventoryrpc.TypeReq, nil, bodyBin)
	if err != nil {
		log.Fatal(err)
	}

	// enc.WriteItem(it)
	// rawMsg := inventoryrpc.EncodeMessage(inventoryrpc.TypeItem, enc.Buf.Bytes())

	msgBuffer := inventoryrpc.PacketBuffer{}
	receivedPktBins, err := msgBuffer.Feed(pktBin)
	if err != nil {
		log.Fatal(err)
	}

	for i := range receivedPktBins {
		fmt.Println("header:")
		fmt.Printf("  length: %v\n", receivedPktBins[i].Length)
		fmt.Printf("  checksum: %v\n", receivedPktBins[i].Checksum)
		fmt.Printf("  id: %v\n", receivedPktBins[i].ID)
		fmt.Printf("  type: %v\n", receivedPktBins[i].Type)
		fmt.Printf("  meta: %v\n", receivedPktBins[i].Meta)

		fmt.Println("body:")
		var receivedBodyMap map[string][]byte
		err := msgpack.Unmarshal(receivedPktBins[i].Body, &receivedBodyMap)
		if err != nil {
			log.Fatal(err)
		}

		for k, v := range receivedBodyMap {
			switch k {
			case "function":
				fmt.Printf("  %s: %s\n", k, string(v))
			case "item":
				var receivedItem inventory.Item
				err := msgpack.Unmarshal(v, &receivedItem)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error decoding item: %v\n", err)
					break
				}
				fmt.Printf("  %s: %+v\n", k, receivedItem)
			default:
				fmt.Printf("  %s: %v\n", k, v)
			}
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
