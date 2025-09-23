package inventorypb

import (
	"database/sql"
	"inventory"
	"inventoryrpc"
	"log"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type Processor struct {
	PacketBuffer      inventoryrpc.PacketBuffer
	PktBeingProcessed map[uuid.UUID]*inventoryrpc.Packet
	Mutex             sync.Mutex
	ProcessingChan    chan *inventoryrpc.Packet
	Db                *sql.DB
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
			inventory.AddItem(p.Db, invItem)
		}
	}
	return nil
}
