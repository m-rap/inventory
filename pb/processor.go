package inventorypb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"inventoryrpc"
	reflect "reflect"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type IProcessor interface {
	Process(data any) (any, error)
}

type PipelineElement struct {
	Processor IProcessor
	Sinks     []*PipelineElement
}

func GetTypeNameFromType(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return "ptr" + t.Elem().Name()
	}
	if t.Kind() == reflect.Slice {
		if t.Elem().Kind() == reflect.Ptr {
			return t.Kind().String() + GetTypeNameFromType(t.Elem())
		}
		return t.Kind().String() + t.Elem().Name()
	}
	return t.Kind().String() + t.Name()
}

func GetTypeName(d any) string {
	if d == nil {
		return "nil"
	}
	t := reflect.TypeOf(d)
	return GetTypeNameFromType(t)
}

func (c *PipelineElement) ProcessThenPass(data any) error {
	//fmt.Printf("enter ProcessThenPass, proc %s in %s\n", GetTypeName(c.Processor), GetTypeName(data))
	processed, err := c.Processor.Process(data)
	if err != nil {
		return err
	}
	//fmt.Printf("out %s\n", GetTypeName(processed))
	for _, s := range c.Sinks {
		err = s.ProcessThenPass(processed)
		if err != nil {
			return err
		}
	}
	return nil
}

func UnmarshalPkt(receivedPktBin []byte) (*inventoryrpc.Packet, error) {
	var receivedPkt Packet
	err := proto.Unmarshal(receivedPktBin, &receivedPkt)

	return ToInvPacket(&receivedPkt), err
}

type PktUnwrapperProcessor struct {
	PacketBuffer inventoryrpc.PacketBuffer
}

func (p *PktUnwrapperProcessor) Process(data any) (any, error) {
	incomingBytes, ok := data.([]byte)
	if !ok {
		return nil, errors.New("invalid PktUnwrapperProcessor input")
	}
	packetWrappers, err := p.PacketBuffer.Feed(incomingBytes)
	if err != nil {
		return nil, err
	}
	receivedPkts := []*inventoryrpc.Packet{}
	for pwIdx := range packetWrappers {
		receivedPktBin := packetWrappers[pwIdx].PacketBytes
		receivedPkt, err := UnmarshalPkt(receivedPktBin)
		if err != nil {
			return nil, err
		}
		receivedPkts = append(receivedPkts, receivedPkt)
	}
	return receivedPkts, nil
}

type ServerProcessorOutput struct {
	Packet  *inventoryrpc.Packet
	Message string
	Code    int32
	Err     error
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
var ErrNoSuchFunc = errors.New("no such func")

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
