package inventorymsgpack

import (
	"inventory"
	"inventoryrpc"

	"github.com/google/uuid"
)

type Account struct {
	UUID             []byte            `msgpack:"uuid,omitempty"`
	Name             string            `msgpack:"name,omitempty"`
	ParentUUID       []byte            `msgpack:"parent,omitempty"`
	TransactionLines []TransactionLine `msgpack:"transaction_lines,omitempty"`
}

type Item struct {
	UUID             []byte            `msgpack:"uuid,omitempty"`
	Name             string            `msgpack:"name,omitempty"`
	Description      string            `msgpack:"description,omitempty"`
	Unit             string            `msgpack:"unit,omitempty"`
	TransactionLines []TransactionLine `msgpack:"transaction_lines,omitempty"`
}

type Transaction struct {
	UUID             []byte            `msgpack:"uuid,omitempty"`
	Description      string            `msgpack:"description,omitempty"`
	DatetimeMs       int64             `msgpack:"date,omitempty"`
	TransactionLines []TransactionLine `msgpack:"transaction_lines,omitempty"`
}

type TransactionLine struct {
	UUID            []byte  `msgpack:"uuid,omitempty"`
	TransactionUUID []byte  `msgpack:"transaction_uuid,omitempty"`
	AccountUUID     []byte  `msgpack:"account_uuid,omitempty"`
	ItemUUID        []byte  `msgpack:"item_uuid,omitempty"`
	Quantity        float64 `msgpack:"quantity,omitempty"`
	Unit            string  `msgpack:"unit,omitempty"`
	Price           float64 `msgpack:"price,omitempty"`
	Currency        string  `msgpack:"currency,omitempty"`
	Note            string  `msgpack:"note,omitempty"`
}

type BalanceHistoryReferences struct {
	TransactionLineUUID []byte `msgpack:"transaction_line,omitempty"`
	TransactionUUID     []byte `msgpack:"transaction_uuid,omitempty"`
	AccountUUID         []byte `msgpack:"account_uuid,omitempty"`
	ItemUUID            []byte `msgpack:"item_uuid,omitempty"`
}

type BalanceHistory struct {
	UUID             []byte                   `msgpack:"uuid,omitempty"`
	Path             []string                 `msgpack:"path,omitempty"`
	References       BalanceHistoryReferences `msgpack:"references,omitempty"`
	Unit             string                   `msgpack:"unit,omitempty"`
	Quantity         float64                  `msgpack:"quantity,omitempty"`
	AvgCost          float64                  `msgpack:"avg_cost,omitempty"`
	Value            float64                  `msgpack:"value,omitempty"`
	DatetimeMs       int64                    `msgpack:"date,omitempty"`
	TransactionPrice float64                  `msgpack:"transaction_price,omitempty"`
	MarketPrice      float64                  `msgpack:"market_price,omitempty"`
	Currency         string                   `msgpack:"currency,omitempty"`
	MarketValue      float64                  `msgpack:"market_value,omitempty"`
	Description      string                   `msgpack:"description,omitempty"`
}

type UnitConversions struct {
	FromUnit   string  `msgpack:"from_unit,omitempty"`
	ToUnit     string  `msgpack:"to_unit,omitempty"`
	Factor     float64 `msgpack:"factor,omitempty"`
	DatetimeMs int64   `msgpack:"date,omitempty"`
}

type CurrencyConversions struct {
	FromCurrency string  `msgpack:"from_currency,omitempty"`
	ToCurrency   string  `msgpack:"to_currency,omitempty"`
	Rate         float64 `msgpack:"rate,omitempty"`
	DatetimeMs   int64   `msgpack:"date,omitempty"`
}

type MarketPrices struct {
	ItemUUID   []byte  `msgpack:"item_uuid,omitempty"`
	DatetimeMs int64   `msgpack:"date,omitempty"`
	Price      float64 `msgpack:"price,omitempty"`
	Unit       string  `msgpack:"unit,omitempty"`
	Currency   string  `msgpack:"currency,omitempty"`
}

type Packet struct {
	UUID []byte            `msgpack:"uuid,omitempty"`
	Type int16             `msgpack:"type,omitempty"`
	Meta map[string][]byte `msgpack:"meta,omitempty"`
	Body map[string][]byte `msgpack:"body,omitempty"`
}

func NewPacket(pkt *inventoryrpc.Packet) *Packet {
	return &Packet{
		UUID: pkt.UUID[:],
		Type: pkt.Type,
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func ToInvPacket(pkt *Packet) *inventoryrpc.Packet {
	pktUUID, _ := uuid.FromBytes(pkt.UUID)
	return &inventoryrpc.Packet{
		UUID: pktUUID,
		Type: pkt.Type,
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
	var lines []TransactionLine
	for i := range transactionLines {
		lines = append(lines, *NewTransactionLine(transactionLines[i]))
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
