package main

import (
	"fmt"
	"inventoryexamplecommon"
	"inventoryrpc"
	"log"
	"net"
	"time"
)

var e inventoryexamplecommon.ExampleProtobuf

func doSend() {
	pktWrapperBin := inventoryexamplecommon.CreatePacketToSend(&e)

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

		if !inventoryexamplecommon.ProcessReceivedPacket(&e, &msgBuffer, recvBuff[:nRecv]) {
			continue
		}
	}
}

func main() {
	go doBind()
	doSend()
}
