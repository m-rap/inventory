package inventorypb

import (
	"inventory"
	"inventoryrpc"

	"github.com/google/uuid"
)

func NewPacket(pkt *inventoryrpc.Packet) *Packet {
	return &Packet{
		UUID: pkt.UUID[:],
		Type: int32(pkt.Type),
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func ToInvPacket(pkt *Packet) *inventoryrpc.Packet {
	pktUUID, _ := uuid.FromBytes(pkt.UUID)
	return &inventoryrpc.Packet{
		UUID: pktUUID,
		Type: int16(pkt.Type),
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func NewTransactionLine(transactionLine inventory.TransactionLine) *TransactionLine {
	trLine := &TransactionLine{
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
	return trLine
}

func NewItem(item *inventory.Item, transactionLines []inventory.TransactionLine) *Item {
	var lines []*TransactionLine
	for i := range transactionLines {
		lines = append(lines, NewTransactionLine(transactionLines[i]))
	}
	return &Item{
		UUID:             item.UUID[:],
		Name:             item.Name,
		Description:      item.Description,
		Unit:             item.Unit,
		TransactionLines: lines,
	}
}

func ToInvItem(item *Item) *inventory.Item {
	itemUUID, _ := uuid.FromBytes(item.UUID)
	return &inventory.Item{
		UUID:        itemUUID,
		Name:        item.Name,
		Description: item.Description,
		Unit:        item.Unit,
	}
}

func ToInvAccount(acc *Account) *inventory.Account {
	accUUID, _ := uuid.FromBytes(acc.UUID)
	parentUUID, _ := uuid.FromBytes(acc.ParentUUID)
	return &inventory.Account{
		UUID: accUUID,
		Name: acc.Name,
		Parent: &inventory.Account{
			UUID: parentUUID,
		},
	}
}

func ToInvTransaction(tr *Transaction) *inventory.Transaction {
	trUUID, _ := uuid.FromBytes(tr.UUID)
	var trLines []*inventory.TransactionLine
	for i := range tr.TransactionLines {
		trl := tr.TransactionLines[i]
		trLineUUID, _ := uuid.FromBytes(trl.UUID)
		accUUID, _ := uuid.FromBytes(trl.AccountUUID)
		itUUID, _ := uuid.FromBytes(trl.ItemUUID)
		trLines = append(trLines, &inventory.TransactionLine{
			UUID: trLineUUID,
			Transaction: &inventory.Transaction{
				UUID: trUUID,
			},
			Account: &inventory.Account{
				UUID: accUUID,
			},
			Item: &inventory.Item{
				UUID: itUUID,
			},
			Quantity: trl.Quantity,
			Unit:     trl.Unit,
			Price:    trl.Price,
			Currency: trl.Currency,
			Note:     trl.Note,
		})
	}
	return &inventory.Transaction{
		UUID:             trUUID,
		Description:      tr.Description,
		DatetimeMs:       tr.DatetimeMs,
		TransactionLines: trLines,
	}
}
