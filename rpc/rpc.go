package inventoryrpc

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"io"
	"log"
	"time"

	"inventory"
)

const (
	RPCAddTransaction byte = 1
	RPCGetBalances    byte = 2
	RPCGetLogs        byte = 3
)

type RPCRequest struct {
	FuncCode byte
	Payload  []byte
}

type RPCResponse struct {
	StatusCode byte
	Payload    []byte
}

func successResp(data []byte) RPCResponse {
	return RPCResponse{StatusCode: 0, Payload: data}
}

func errorResp(msg string) RPCResponse {
	return RPCResponse{StatusCode: 1, Payload: []byte(msg)}
}

func ReadRPCRequest(r io.Reader) (RPCRequest, error) {
	var code byte
	var length uint32

	if err := binary.Read(r, binary.LittleEndian, &code); err != nil {
		return RPCRequest{}, err
	}
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return RPCRequest{}, err
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return RPCRequest{}, err
	}

	return RPCRequest{FuncCode: code, Payload: payload}, nil
}

func WriteRPCResponse(w io.Writer, resp RPCResponse) error {
	if err := binary.Write(w, binary.LittleEndian, resp.StatusCode); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(resp.Payload))); err != nil {
		return err
	}
	_, err := w.Write(resp.Payload)
	return err
}

func HandleRPC(invs map[string]*inventory.Inventory, db *sql.DB, req RPCRequest) (RPCResponse, error) {
	buf := bytes.NewReader(req.Payload)
	invID, err := readString(buf)
	if err != nil {
		return errorResp("missing inventory ID"), nil
	}

	inv, ok := invs[invID]
	if !ok {
		return errorResp("inventory not found"), nil
	}

	switch req.FuncCode {
	case RPCAddTransaction:
		tx, err := decodeTransaction(buf)
		if err != nil {
			return errorResp("invalid transaction payload"), nil
		}
		tx.InventoryID = invID
		inv.AddTransaction(tx.Type, tx.Items, tx.Note)
		if err := inv.SaveTransactionToDB(db, invID, tx); err != nil {
			log.Println("DB save error:", err)
		}
		return successResp(nil), nil

	case RPCGetBalances:
		balances := inv.GetBalances()
		data, err := encodeBalances(balances)
		if err != nil {
			return errorResp("encode failed"), nil
		}
		return successResp(data), nil

	case RPCGetLogs:
		logs := inv.GetLogs()
		data, err := encodeLogs(logs)
		if err != nil {
			return errorResp("encode failed"), nil
		}
		return successResp(data), nil

	default:
		return errorResp("unknown function"), nil
	}
}

// Binary encoding/decoding helpers (simplified)
func decodeTransaction(r io.Reader) (inventory.Transaction, error) {
	var tx inventory.Transaction

	id, err := readString(r)
	if err != nil {
		return tx, err
	}
	tx.ID = id
	if err := binary.Read(r, binary.LittleEndian, &tx.Type); err != nil {
		return tx, err
	}
	var ts int64
	if err := binary.Read(r, binary.LittleEndian, &ts); err != nil {
		return tx, err
	}
	tx.Timestamp = time.Unix(0, ts)
	note, err := readString(r)
	if err != nil {
		return tx, err
	}
	tx.Note = note

	var count uint16
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return tx, err
	}

	tx.Items = make([]inventory.TransactionItem, count)
	for i := range tx.Items {
		itemID, _ := readString(r)
		var qty, balance int32
		var price float64
		unit, _ := readString(r)
		currency, _ := readString(r)
		binary.Read(r, binary.LittleEndian, &qty)
		binary.Read(r, binary.LittleEndian, &balance)
		binary.Read(r, binary.LittleEndian, &price)

		tx.Items[i] = inventory.TransactionItem{
			ItemID:    itemID,
			Quantity:  int(qty),
			Unit:      unit,
			Balance:   int(balance),
			UnitPrice: price,
			Currency:  currency,
		}
	}

	return tx, nil
}

func encodeBalances(m map[string]int) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(m))); err != nil {
		return nil, err
	}
	for k, v := range m {
		if err := writeString(buf, k); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, int32(v)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func encodeLogs(logs []string) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(logs))); err != nil {
		return nil, err
	}
	for _, line := range logs {
		if err := writeString(buf, line); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func writeString(w io.Writer, s string) error {
	if err := binary.Write(w, binary.LittleEndian, uint16(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func readString(r io.Reader) (string, error) {
	var length uint16
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	buf := make([]byte, length)
	_, err := io.ReadFull(r, buf)
	return string(buf), err
}
