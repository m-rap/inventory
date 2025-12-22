package inventorypb

import (
	"inventory"
	"inventoryrpc"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type ServerProcessor struct {
	ProcessingChan                 chan *inventoryrpc.Packet
	ConsumeProcessingResponseFuncs []ConsumeProcessingResponseFunc
}

func NewServerProcessor() *ServerProcessor {
	return &ServerProcessor{
		ProcessingChan: make(chan *inventoryrpc.Packet),
	}
}

func (p *ServerProcessor) ProcessPkt(pkt *inventoryrpc.Packet) (*inventoryrpc.Packet, string, int32, error) {
	// layer 0, check func
	funcBytes, ok := pkt.Body["function"]
	if !ok {
		return CreateRespPkt(pkt.UUID, -201, nil, ErrReqHasNoFunc, ErrReqHasNoFunc.Error())
	}

	funcStr := string(funcBytes)
	var payload map[string][]byte
	argBytes, argOk := pkt.Body["arg"]

	// layer 1, check curr db
	switch funcStr {
	case "GetCurrDB", "AddItem", "AddAccount", "ApplyTransaction", "GetMainAccounts":
		if inventory.CurrDB == nil {
			return CreateRespPkt(pkt.UUID, -202, nil, ErrCurrDbNil, ErrCurrDbNil.Error())
		}
	}

	// layer 2, check arg ok
	switch funcStr {
	case "AddItem", "AddAccount", "ApplyTransaction":
		if !argOk {
			return CreateRespPkt(pkt.UUID, -203, nil, ErrReqHasNoArg, ErrCurrDbNil.Error())
		}
	}

	var err error

	// layer last
	switch funcStr {
	case "GetCurrDB":
		if err != nil {
			return CreateRespPktErrExecFunc(pkt.UUID, err)
		}
		var dbUUIDBytes []byte = nil
		for elDbUUID, db := range inventory.DBMap {
			if db == inventory.CurrDB {
				dbUUIDBytes = elDbUUID[:]
				payload["uuid"] = dbUUIDBytes
			}
		}
		if dbUUIDBytes == nil {
			return CreateRespPktErrExecFunc(pkt.UUID, ErrCurrDbNotRegistered)
		}
	case "OpenOrCreateDB":
		var dbUUID uuid.UUID
		if argOk {
			dbUUID, err = uuid.FromBytes(argBytes)
			if err != nil {
				return CreateRespPktErrUnmarshall(pkt.UUID, err)
			}
		} else {
			dbUUID, err = uuid.NewV6()
			if err != nil {
				return CreateRespPktErrExecFunc(pkt.UUID, err)
			}
		}
		_, err = inventory.OpenOrCreateDB(dbUUID)
		if err != nil {
			return CreateRespPktErrExecFunc(pkt.UUID, err)
		}
		payload["uuid"] = dbUUID[:]
	case "AddItem":
		var item Item
		err = proto.Unmarshal(argBytes, &item)
		if err != nil {
			return CreateRespPktErrUnmarshall(pkt.UUID, err)
		}
		invItem := ToInvItem(&item)
		entityUUIDBytes, err := inventory.AddItem(inventory.CurrDB, invItem)
		if err != nil {
			return CreateRespPktErrExecFunc(pkt.UUID, err)
		}
		payload["uuid"] = entityUUIDBytes
	case "AddAccount":
		var acc Account
		err = proto.Unmarshal(argBytes, &acc)
		if err != nil {
			return CreateRespPktErrUnmarshall(pkt.UUID, err)
		}
		invAcc := ToInvAccount(&acc)
		entityUUIDBytes, err := inventory.AddAccount(inventory.CurrDB, invAcc)
		if err != nil {
			return CreateRespPktErrExecFunc(pkt.UUID, err)
		}
		payload["uuid"] = entityUUIDBytes
	case "ApplyTransaction":
		var tr Transaction
		err = proto.Unmarshal(argBytes, &tr)
		if err != nil {
			return CreateRespPktErrUnmarshall(pkt.UUID, err)
		}
		invTr := ToInvTransaction(&tr)
		entityUUIDBytes, err := inventory.ApplyTransaction(inventory.CurrDB, invTr)
		if err != nil {
			return CreateRespPktErrExecFunc(pkt.UUID, err)
		}
		payload["uuid"] = entityUUIDBytes
	case "GetMainAccounts":
		accs := []*inventory.Account{
			inventory.AssetAcc,
			inventory.EquityAcc,
			inventory.LiabilityAcc,
			inventory.IncomeAcc,
			inventory.ExpenseAcc,
		}
		for _, a := range accs {
			payload[a.Name] = a.UUID[:]
		}
	}

	return CreateRespPkt(pkt.UUID, 0, payload, nil, "ok")
}

func (p *ServerProcessor) PostProcessPkt(responsePkt *inventoryrpc.Packet) error {
	responsePktPb := NewPacket(responsePkt)
	responsePktByte, err := proto.Marshal(responsePktPb)
	if err != nil {
		return err
	}
	for _, consumeFunc := range p.ConsumeProcessingResponseFuncs {
		consumeFunc(responsePktByte)
	}
	return nil
}

// func (p *ServerProcessor) Process() error {
// 	for pkt := range p.ProcessingChan {
// 		_, err := p.ProcessPkt(pkt)
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "error processing: %v\n", err.Error())
// 		}
// 	}
// 	return nil
// }
