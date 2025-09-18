package main

import (
	"fmt"
	"inventory"
	"inventorymsgpack"
	"inventorypb"
	"inventoryrpc"
	"log"
	"os"

	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

type ExampleInterface interface {
	OnMarshalItem(item *inventory.Item) ([]byte, error)
	OnMarshalPkt(pkt *inventoryrpc.Packet) ([]byte, error)
	OnUnmarshalPkt(receivedPktBin []byte) (inventoryrpc.Packet, error)
	OnUnmarshalItem(receivedItemBin []byte) (inventory.Item, error)
}

type ExampleMsgpack struct{}

func (e *ExampleMsgpack) OnMarshalItem(item *inventory.Item) ([]byte, error) {
	// Example encode/decode roundtrip
	// enc := inventoryrpc.NewEncoder()
	it := inventorymsgpack.NewItem(item)
	// Marshal to bytes
	return msgpack.Marshal(&it)
}

func (e *ExampleMsgpack) OnMarshalPkt(pkt *inventoryrpc.Packet) ([]byte, error) {
	msgpackPkt := inventorymsgpack.NewPacket(pkt)
	return msgpack.Marshal(&msgpackPkt)
}

func (e *ExampleMsgpack) OnUnmarshalPkt(receivedPktBin []byte) (inventoryrpc.Packet, error) {
	var receivedPkt inventorymsgpack.Packet
	err := msgpack.Unmarshal(receivedPktBin, &receivedPkt)

	return inventorymsgpack.ToInvPacket(&receivedPkt), err
}

func (e *ExampleMsgpack) OnUnmarshalItem(receivedItemBin []byte) (inventory.Item, error) {
	var receivedItem inventorymsgpack.Item
	err := msgpack.Unmarshal(receivedItemBin, &receivedItem)
	return inventorymsgpack.ToInvItem(&receivedItem), err
}

type ExampleProtobuf struct{}

func (e *ExampleProtobuf) OnMarshalItem(item *inventory.Item) ([]byte, error) {
	// Example encode/decode roundtrip
	// enc := inventoryrpc.NewEncoder()
	it := inventorypb.NewItem(item)
	// Marshal to bytes
	return proto.Marshal(&it)
}

func (e *ExampleProtobuf) OnMarshalPkt(pkt *inventoryrpc.Packet) ([]byte, error) {
	pbPkt := inventorypb.NewPacket(pkt)
	return proto.Marshal(&pbPkt)
}

func (e *ExampleProtobuf) OnUnmarshalPkt(receivedPktBin []byte) (inventoryrpc.Packet, error) {
	var receivedPkt inventorypb.Packet
	err := proto.Unmarshal(receivedPktBin, &receivedPkt)

	return inventorypb.ToInvPacket(&receivedPkt), err
}

func (e *ExampleProtobuf) OnUnmarshalItem(receivedItemBin []byte) (inventory.Item, error) {
	var receivedItem inventorypb.Item
	err := proto.Unmarshal(receivedItemBin, &receivedItem)
	return inventorypb.ToInvItem(&receivedItem), err
}

func doExample(e ExampleInterface) []byte {
	it := inventory.Item{UUID: "I1", Name: "Steel", Description: "Raw material", Unit: "kg"}
	itemBin, err := e.OnMarshalItem(&it)
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

	pktBin, err := e.OnMarshalPkt(&pkt)
	if err != nil {
		log.Fatal(err)
	}

	pktWrapperBin, err := inventoryrpc.EncodePacketWrapper(pktBin)
	if err != nil {
		log.Fatal(err)
	}

	msgBuffer := inventoryrpc.PacketBuffer{}
	receivedPktWrapperBins, err := msgBuffer.Feed(pktWrapperBin)
	if err != nil {
		log.Fatal(err)
	}

	for i := range receivedPktWrapperBins {
		receivedPktBin := receivedPktWrapperBins[i].PacketBytes

		receivedPkt, err := e.OnUnmarshalPkt(receivedPktBin)
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
				receivedItem, err := e.OnUnmarshalItem(v)
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

	return pktWrapperBin
}

func main() {
	exMsgpack := ExampleMsgpack{}
	exProtobuf := ExampleProtobuf{}
	mpBytes := doExample(&exMsgpack)
	pbBytes := doExample(&exProtobuf)
	fmt.Printf("msgpack len %d\n", len(mpBytes))
	fmt.Printf("protobuf len %d\n", len(pbBytes))
}
