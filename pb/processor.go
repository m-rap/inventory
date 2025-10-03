package inventorypb

import (
	"fmt"
	"inventory"
	"inventoryrpc"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type Processor struct {
	PacketBuffer      inventoryrpc.PacketBuffer
	PktBeingProcessed map[uuid.UUID]*inventoryrpc.Packet
	Mutex             sync.Mutex
	ProcessingChan    chan *inventoryrpc.Packet
}

func NewProcessor() *Processor {
	return &Processor{
		ProcessingChan: make(chan *inventoryrpc.Packet),
	}
}

func UnmarshalPkt(receivedPktBin []byte) (*inventoryrpc.Packet, error) {
	var receivedPkt Packet
	err := proto.Unmarshal(receivedPktBin, &receivedPkt)

	return ToInvPacket(&receivedPkt), err
}

func (p *Processor) HandleIncoming(incomingBytes []byte) error {
	packetWrappers, err := p.PacketBuffer.Feed(incomingBytes)
	if err != nil {
		return err
	}
	for pwIdx := range packetWrappers {
		receivedPktBin := packetWrappers[pwIdx].PacketBytes
		receivedPkt, err := UnmarshalPkt(receivedPktBin)
		if err != nil {
			log.Fatal(err)
		}
		p.Mutex.Lock()
		p.PktBeingProcessed[receivedPkt.UUID] = receivedPkt
		p.Mutex.Unlock()
	}
	return nil
}

func (p *Processor) Process() error {
	for pkt := range p.ProcessingChan {
		funcBytes, ok := pkt.Body["function"]
		if !ok {
			continue
		}
		funcStr := string(funcBytes)
		switch funcStr {
		case "OpenOrCreateDB":
			dbUUIDBytes, ok := pkt.Body["dbUUID"]
			if !ok {
				break
			}
			dbUUID, err := uuid.FromBytes(dbUUIDBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting bytes to uuid: %s\n", err.Error())
				break
			}
			inventory.OpenOrCreateDB(dbUUID)
		case "SelectDB":
			dbUUIDBytes, ok := pkt.Body["DbUUID"]
			if !ok {
				break
			}
			dbUUID, err := uuid.FromBytes(dbUUIDBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting bytes to uuid: %s\n", err.Error())
				break
			}
			inventory.SelectDB(dbUUID)
		case "AddItem":
			itemBytes, ok := pkt.Body["Item"]
			if !ok {
				break
			}
			var item Item
			err := proto.Unmarshal(itemBytes, &item)
			if err != nil {
				return err
			}
			invItem := ToInvItem(&item)
			inventory.AddItem(inventory.CurrDB, invItem)
		case "AddAccount":
			accBytes, ok := pkt.Body["Account"]
			if !ok {
				break
			}
			var acc Account
			err := proto.Unmarshal(accBytes, &acc)
			if err != nil {
				return err
			}
			invAcc := ToInvAccount(&acc)
			inventory.AddAccount(inventory.CurrDB, invAcc)
		case "ApplyTransaction":
			trBytes, ok := pkt.Body["Transaction"]
			if !ok {
				break
			}
			var tr Transaction
			err := proto.Unmarshal(trBytes, &tr)
			if err != nil {
				return err
			}
			invTr := ToInvTransaction(&tr)
			inventory.ApplyTransaction(inventory.CurrDB, invTr)
		}
	}
	return nil
}
