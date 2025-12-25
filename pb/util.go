package inventorypb

import (
	"inventory"
	"inventoryrpc"
	"log"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
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

func NewTransactionLine(transactionLine *inventory.TransactionLine) *TransactionLine {
	trLine := &TransactionLine{
		UUID:     transactionLine.UUID[:],
		Quantity: transactionLine.Quantity.Data,
		Unit:     transactionLine.Unit,
		Price:    transactionLine.Price.Data,
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

func NewTransaction(transaction *inventory.Transaction) *Transaction {
	var lines []*TransactionLine
	for i := range transaction.TransactionLines {
		lines = append(lines, NewTransactionLine(transaction.TransactionLines[i]))
	}
	return &Transaction{
		UUID:             transaction.UUID[:],
		Description:      transaction.Description,
		DatetimeMs:       transaction.DatetimeMs,
		TransactionLines: lines,
	}
}

func NewItem(item *inventory.Item, transactionLines []*inventory.TransactionLine) *Item {
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

func NewAccount(account *inventory.Account, transactionLines []*inventory.TransactionLine) *Account {
	var lines []*TransactionLine
	for i := range transactionLines {
		lines = append(lines, NewTransactionLine(transactionLines[i]))
	}
	var parentUUID []byte
	if account.Parent != nil {
		parentUUID = account.Parent.UUID[:]
	} else {
		parentUUID = nil
	}
	return &Account{
		UUID:             account.UUID[:],
		Name:             account.Name,
		ParentUUID:       parentUUID,
		TransactionLines: lines,
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
		var acc *inventory.Account
		if accUUID != uuid.Nil {
			acc = &inventory.Account{
				UUID: accUUID,
			}
		} else {
			acc = nil
		}
		var item *inventory.Item
		if itUUID != uuid.Nil {
			item = &inventory.Item{
				UUID: itUUID,
			}
		} else {
			item = nil
		}
		trLines = append(trLines, &inventory.TransactionLine{
			UUID: trLineUUID,
			Transaction: &inventory.Transaction{
				UUID: trUUID,
			},
			Account:  acc,
			Item:     item,
			Quantity: inventory.NewDecimal(trl.Quantity),
			Unit:     trl.Unit,
			Price:    inventory.NewDecimal(trl.Price),
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

func NewMarketPrice(p *inventory.MarketPrice) *MarketPrice {
	var itemUUID []byte
	if p.Item != nil {
		itemUUID = p.Item.UUID[:]
	} else {
		itemUUID = nil
	}
	return &MarketPrice{
		ItemUUID:   itemUUID,
		DatetimeMs: p.DatetimeMs,
		Price:      p.Price.Data,
		Unit:       p.Unit,
		Currency:   p.Currency,
	}
}

func ToInvMarketPrice(p *MarketPrice) *inventory.MarketPrice {
	itemUUID, _ := uuid.FromBytes(p.ItemUUID)
	return &inventory.MarketPrice{
		Item: &inventory.Item{
			UUID: itemUUID,
		},
		DatetimeMs: p.DatetimeMs,
		Price:      inventory.NewDecimal(p.Price),
		Unit:       p.Unit,
		Currency:   p.Currency,
	}
}

func NewMapOfBytes(m map[string][]byte) *MapOfBytes {
	return &MapOfBytes{
		Content: m,
	}
}

func CreateRequest(reqFunc string, params proto.Message) (uuid.UUID, []byte, error) {
	paramBin, err := proto.Marshal(params)
	if err != nil {
		return uuid.UUID{}, nil, err
	}

	pktUUID, err := uuid.NewV6()
	if err != nil {
		log.Fatal(err)
	}

	pktBin, err := proto.Marshal(&Packet{
		UUID: pktUUID[:],
		Type: inventoryrpc.TypeReq,
		Meta: nil,
		Body: map[string][]byte{
			"function": []byte(reqFunc),
			"arg":      paramBin,
		},
	})

	return pktUUID, pktBin, err
}
