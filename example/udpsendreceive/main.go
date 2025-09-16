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
	pkt := inventoryrpc.Packet{}
	pkt.H["id"] = []byte{0}
	pkt.H["type"] = []byte("reqget")
	pkt.B["item"] = itemBin

	pktBin, err := msgpack.Marshal(&pkt)
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
		for k, v := range receivedPktBins[i].H {
			switch k {
			case "id":
				var id int
				err := msgpack.Unmarshal(v, &id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error decoding id: %v\n", err)
					break
				}
				fmt.Printf("  %s: %d\n", k, id)

			case "type":
				var typ string
				err := msgpack.Unmarshal(v, &typ)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error decoding type: %v\n", err)
					break
				}
				fmt.Printf("  %s: %s\n", k, typ)
			default:
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
		fmt.Println("body:")
		for k, v := range receivedPktBins[i].B {
			switch k {
			case "item":
				var it2 inventory.Item
				err := msgpack.Unmarshal(v, &it2)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error decoding item: %v\n", err)
					break
				}
				fmt.Printf("  %s: %+v\n", k, it2)
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
