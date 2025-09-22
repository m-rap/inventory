package inventorypb

import (
	"inventory"
	"inventoryrpc"

	"github.com/google/uuid"
)

func NewPacket(pkt *inventoryrpc.Packet) Packet {
	return Packet{
		UUID: pkt.UUID[:],
		Type: int32(pkt.Type),
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func ToInvPacket(pkt *Packet) inventoryrpc.Packet {
	pktUUID, _ := uuid.FromBytes(pkt.UUID)
	return inventoryrpc.Packet{
		UUID: pktUUID,
		Type: int16(pkt.Type),
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func NewTransactionLine(transactionLine inventory.TransactionLine) *TransactionLine {
	trLine := TransactionLine{
		UUID:     transactionLine.UUID[:],
		Quantity: transactionLine.Quantity,
		Unit:     transactionLine.Unit,
		Price:    transactionLine.Price,
		Currency: transactionLine.Currency,
		Note:     transactionLine.Note,
	}
	if transactionLine.Transaction != nil {
		trLine.TransactionUUID = transactionLine.Transaction.UUID[:]
	}
	if transactionLine.Account != nil {
		trLine.AccountUUID = transactionLine.Account.UUID[:]
	}
	if transactionLine.Item != nil {
		trLine.ItemUUID = transactionLine.Item.UUID[:]
	}
	return &trLine
}

func NewItem(item *inventory.Item, transactionLines []inventory.TransactionLine) Item {
	var lines []*TransactionLine
	for i := range transactionLines {
		lines = append(lines, NewTransactionLine(transactionLines[i]))
	}
	return Item{
		UUID:             item.UUID[:],
		Name:             item.Name,
		Description:      item.Description,
		Unit:             item.Unit,
		TransactionLines: lines,
	}
}

func ToInvItem(item *Item) inventory.Item {
	itemUUID, _ := uuid.FromBytes(item.UUID)
	return inventory.Item{
		UUID:        itemUUID,
		Name:        item.Name,
		Description: item.Description,
		Unit:        item.Unit,
	}
}
