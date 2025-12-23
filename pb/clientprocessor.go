package inventorypb

import (
	"inventoryrpc"

	"github.com/google/uuid"
)

type ClientProcessorExample struct {
	ResponsePktMap map[uuid.UUID]*inventoryrpc.Packet
}

func NewClientProcessorExample() *ClientProcessorExample {
	return &ClientProcessorExample{
		ResponsePktMap: map[uuid.UUID]*inventoryrpc.Packet{},
	}
}

func (p *ClientProcessorExample) ProcessPkt(pkt *inventoryrpc.Packet) (*inventoryrpc.Packet, string, int32, error) {
	p.ResponsePktMap[pkt.UUID] = pkt
	return nil, "", 0, nil
}

func (p *ClientProcessorExample) PostProcessPkt(responsePkt *inventoryrpc.Packet) error {
	return nil
}
