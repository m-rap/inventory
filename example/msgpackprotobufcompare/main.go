package main

import (
	"fmt"
	"inventoryrpc"

	"inventoryexamplecommon"
)

func doExample(e inventoryexamplecommon.ExampleInterface) []byte {
	pktWrapperBin := inventoryexamplecommon.CreatePacketToSend(e)

	msgBuffer := inventoryrpc.PacketBuffer{}
	inventoryexamplecommon.ProcessReceivedPacket(e, &msgBuffer, pktWrapperBin)

	return pktWrapperBin
}

func main() {
	exMsgpack := inventoryexamplecommon.ExampleMsgpack{}
	exProtobuf := inventoryexamplecommon.ExampleProtobuf{}
	mpBytes := doExample(&exMsgpack)
	pbBytes := doExample(&exProtobuf)
	fmt.Printf("msgpack len %d\n", len(mpBytes))
	fmt.Printf("protobuf len %d\n", len(pbBytes))
}
