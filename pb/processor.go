package inventorypb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"inventoryrpc"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func UnmarshalPkt(receivedPktBin []byte) (*inventoryrpc.Packet, error) {
	var receivedPkt Packet
	err := proto.Unmarshal(receivedPktBin, &receivedPkt)

	return ToInvPacket(&receivedPkt), err
}

// type ResponsePktBody struct {
// 	UUID    uuid.UUID
// 	Message string
// 	Code    int32
// }

// func (pkt *ResponsePktBody) ToMap() map[string][]byte {
// 	codeByte := make([]byte, 4)
// 	binary.LittleEndian.PutUint32(codeByte, uint32(pkt.Code))
// 	return map[string][]byte{
// 		"message": []byte(pkt.Message),
// 		"code":    codeByte,
// 	}
// }

// func (pkt *ResponsePktBody) FromMap(m map[string][]byte) {
// 	msgByte, ok := m["message"]
// 	if ok {
// 		pkt.Message = string(msgByte)
// 	}
// 	codeByte, ok := m["code"]
// 	if ok {
// 		pkt.Code = int32(binary.LittleEndian.Uint32(codeByte))
// 	}
// }

var ErrNoProcessor = errors.New("no processor")
var ErrCurrDbNotRegistered = errors.New("curr db not registered")
var ErrReqHasNoFunc = errors.New("request has no function")
var ErrCurrDbNil = errors.New("curr db nil")
var ErrReqHasNoArg = errors.New("request has no arg")

// returns packet, message, code, error
func CreateRespPkt(UUID uuid.UUID, code int32, payload map[string][]byte, err error, format string, a ...any) (*inventoryrpc.Packet, string, int32, error) {
	msg := fmt.Sprintf(format, a...)
	// respPktBody := &ResponsePktBody{
	// 	UUID:    UUID,
	// 	Message: msg,
	// 	Code:    code,
	// }
	codeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(codeBytes, uint32(code))

	pkt := &inventoryrpc.Packet{
		UUID: UUID,
		Type: inventoryrpc.TypeResp,
		Meta: nil,
		Body: map[string][]byte{
			"code":    codeBytes,
			"message": []byte(msg),
		},
	}
	for k, v := range payload {
		pkt.Body[k] = v
	}
	return pkt, msg, code, err
}

func CreateRespPktErrUnmarshall(UUID uuid.UUID, err error) (*inventoryrpc.Packet, string, int32, error) {
	return CreateRespPkt(UUID, -101, nil, err, "error unmarshall: %s", err.Error())
}

func CreateRespPktErrExecFunc(UUID uuid.UUID, err error) (*inventoryrpc.Packet, string, int32, error) {
	return CreateRespPkt(UUID, -102, nil, err, "error execute function: %s", err.Error())
}

type ConsumeProcessingResponseFunc func(responsePktByte []byte)

type ProcessorInterface interface {
	ProcessPkt(pkt *inventoryrpc.Packet) (*inventoryrpc.Packet, string, int32, error)
	PostProcessPkt(responsePkt *inventoryrpc.Packet) error
}

type PacketReceiver struct {
	PacketBuffer      inventoryrpc.PacketBuffer
	PktBeingProcessed map[uuid.UUID]*inventoryrpc.Packet
	Mutex             sync.Mutex
	Processor         ProcessorInterface
}

func (p *PacketReceiver) HandleIncoming(incomingBytes []byte) error {
	if p.Processor == nil {
		return ErrNoProcessor
	}
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

		go func() {
			responsePkt, message, _, err := p.Processor.ProcessPkt(receivedPkt)
			if err != nil {
				fmt.Fprintf(os.Stderr, message)
			}
			err = p.Processor.PostProcessPkt(responsePkt)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}

			p.Mutex.Lock()
			delete(p.PktBeingProcessed, receivedPkt.UUID)
			p.Mutex.Unlock()
		}()
	}
	return nil
}
