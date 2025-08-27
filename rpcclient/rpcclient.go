package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"inventory"
	"inventoryrpc"
	"time"
)

// Helper to build a request for balances
func buildGetBalancesRequest(invID string) inventoryrpc.RPCRequest {
	buf := new(bytes.Buffer)
	_ = writeString(buf, invID)
	return inventoryrpc.RPCRequest{
		FuncCode: inventoryrpc.RPCGetBalances,
		Payload:  buf.Bytes(),
	}
}

// Client-side decodeBalances (mirror of encodeBalances)
func decodeBalances(r *bytes.Reader) (map[string]inventory.Balance, error) {
	var count uint16
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}

	balances := make(map[string]inventory.Balance, count)
	for i := 0; i < int(count); i++ {
		key, err := readString(r)
		if err != nil {
			return nil, err
		}

		var qty int32
		var value float64
		if err := binary.Read(r, binary.LittleEndian, &qty); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			return nil, err
		}

		unit, err := readString(r)
		if err != nil {
			return nil, err
		}
		currency, err := readString(r)
		if err != nil {
			return nil, err
		}

		balances[key] = inventory.Balance{
			Quantity: int(qty),
			Value:    value,
			Unit:     unit,
			Currency: currency,
		}
	}
	return balances, nil
}

func writeString(w *bytes.Buffer, s string) error {
	if err := binary.Write(w, binary.LittleEndian, uint16(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func readString(r *bytes.Reader) (string, error) {
	var length uint16
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := r.Read(buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func main() {
	// Demo: simulate balances request/response locally without network
	inv := inventory.NewInventory("root")
	appleID := inventory.GenerateUUID()
	inv.RegisterItem(inventory.Item{
		ID:       appleID,
		Name:     "Apple",
		Unit:     "pcs",
		Currency: "IDR",
	})

	inv.AddItems([]inventory.TransactionItem{
		{ItemID: appleID, Quantity: 2, Unit: "pcs", UnitPrice: 1000, Currency: "IDR"},
	}, "init", time.Now())

	// Build request
	req := buildGetBalancesRequest(inv.ID)

	// Simulate server handle
	invs := map[string]*inventory.Inventory{"root": inv}
	resp, _ := inventoryrpc.HandleRPC(invs, nil, req)

	// Decode response
	buf := bytes.NewReader(resp.Payload)
	balances, err := decodeBalances(buf)
	if err != nil {
		panic(err)
	}

	// Print result
	for id, bal := range balances {
		fmt.Printf("%s: %d %s (Value %.2f %s)\n", id, bal.Quantity, bal.Unit, bal.Value, bal.Currency)
	}
}
