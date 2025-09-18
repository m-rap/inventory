package main

import (
	"fmt"
	"inventory"
	"inventorypb"
	"inventoryrpc"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/protobuf/proto"
)

func OnMarshalItem(item *inventory.Item) ([]byte, error) {
	// Example encode/decode roundtrip
	// enc := inventoryrpc.NewEncoder()
	it := inventorypb.NewItem(item)
	// Marshal to bytes
	return proto.Marshal(&it)
}

func OnMarshalPkt(pkt *inventoryrpc.Packet) ([]byte, error) {
	pbPkt := inventorypb.NewPacket(pkt)
	return proto.Marshal(&pbPkt)
}

func OnUnmarshalPkt(receivedPktBin []byte) (inventoryrpc.Packet, error) {
	var receivedPkt inventorypb.Packet
	err := proto.Unmarshal(receivedPktBin, &receivedPkt)

	return inventorypb.ToInvPacket(&receivedPkt), err
}

func OnUnmarshalItem(receivedItemBin []byte) (inventory.Item, error) {
	var receivedItem inventorypb.Item
	err := proto.Unmarshal(receivedItemBin, &receivedItem)
	return inventorypb.ToInvItem(&receivedItem), err
}

func doSend() {
	it := inventory.Item{UUID: "I1", Name: "Steel", Description: "Raw material", Unit: "kg"}
	itemBin, err := OnMarshalItem(&it)
	if err != nil {
		log.Fatal(err)
	}

	pkt := inventoryrpc.Packet{
		ID:   0,
		Type: inventoryrpc.TypeReq,
		Meta: nil,
		Body: map[string][]byte{
			"function": []byte("InsertItem"),
			"item":     itemBin,
		},
	}

	pktBin, err := OnMarshalPkt(&pkt)
	if err != nil {
		log.Fatal(err)
	}

	pktWrapperBin, err := inventoryrpc.EncodePacketWrapper(pktBin)
	if err != nil {
		log.Fatal(err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:12001")
	if err != nil {
		log.Fatal(err.Error())
	}

	udpsender, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatal(err.Error())
	}

	halfN := len(pktWrapperBin) / 2
	udpsender.Write(pktWrapperBin[0:halfN])
	time.Sleep(1000 * time.Millisecond)
	udpsender.Write(pktWrapperBin[halfN:])
	time.Sleep(1000 * time.Millisecond)

	defer udpsender.Close()
}

func doBind() {
	serverAddr, err := net.ResolveUDPAddr("udp", ":12001")
	if err != nil {
		log.Fatal(err.Error())
	}
	udpreceiver, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		panic("can't create udp")
	}
	defer udpreceiver.Close()

	msgBuffer := inventoryrpc.PacketBuffer{}

	for {
		recvBuff := make([]byte, 4096)
		nRecv, remoteAddr, err := udpreceiver.ReadFromUDP(recvBuff)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("received %d from %v\n", nRecv, remoteAddr)

		receivedPktWrapperBins, err := msgBuffer.Feed(recvBuff[:nRecv])
		if err != nil {
			log.Fatal(err)
		}
		if len(receivedPktWrapperBins) == 0 {
			continue
		}

		for i := range receivedPktWrapperBins {
			receivedPktBin := receivedPktWrapperBins[i].PacketBytes

			receivedPkt, err := OnUnmarshalPkt(receivedPktBin)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("header:")
			fmt.Printf("  length: %v\n", receivedPktWrapperBins[i].Length)
			fmt.Printf("  checksum: %v\n", receivedPktWrapperBins[i].Checksum)
			fmt.Printf("  id: %v\n", receivedPkt.ID)
			fmt.Printf("  type: %v\n", receivedPkt.Type)
			fmt.Printf("  meta: %v\n", receivedPkt.Meta)

			fmt.Println("body:")
			for k, v := range receivedPkt.Body {
				switch k {
				case "function":
					fmt.Printf("  %s: %s\n", k, string(v))
				case "item":
					receivedItem, err := OnUnmarshalItem(v)
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
	}
}

func main() {
	go doBind()
	doSend()
}
