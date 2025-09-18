package inventorymsgpack

import (
	"inventory"
	"inventoryrpc"
)

type Item struct {
	ID          int    `msgpack:"id,omitempty"`
	UUID        string `msgpack:"uuid,omitempty"`
	Name        string `msgpack:"name,omitempty"`
	Description string `msgpack:"description,omitempty"`
	Unit        string `msgpack:"unit,omitempty"`
}

type Transaction struct {
	ID          int               `msgpack:"id,omitempty"`
	UUID        string            `msgpack:"uuid,omitempty"`
	Description string            `msgpack:"description,omitempty"`
	DatetimeMs  int64             `msgpack:"date,omitempty"`
	Year        int               `msgpack:"year,omitempty"`
	Month       uint8             `msgpack:"month,omitempty"`
	Lines       []TransactionLine `msgpack:"lines,omitempty"`
}

type TransactionLine struct {
	ID              int     `msgpack:"id,omitempty"`
	TransactionUUID string  `msgpack:"transaction_uuid,omitempty"`
	AccountUUID     string  `msgpack:"account_uuid,omitempty"`
	ItemUUID        string  `msgpack:"item_uuid,omitempty"`
	Quantity        float64 `msgpack:"quantity,omitempty"`
	Unit            string  `msgpack:"unit,omitempty"`
	Price           float64 `msgpack:"price,omitempty"`
	Currency        string  `msgpack:"currency,omitempty"`
	Note            string  `msgpack:"note,omitempty"`
}

type BalanceHistory struct {
	ID          int      `msgpack:"id,omitempty"`
	Path        []string `msgpack:"path,omitempty"`
	AccountUUID string   `msgpack:"account_uuid,omitempty"`
	ItemUUID    string   `msgpack:"item_uuid,omitempty"`
	Unit        string   `msgpack:"unit,omitempty"`
	Quantity    float64  `msgpack:"quantity,omitempty"`
	AvgCost     float64  `msgpack:"avg_cost,omitempty"`
	Value       float64  `msgpack:"value,omitempty"`
	DatetimeMs  int64    `msgpack:"date,omitempty"`
	Year        int      `msgpack:"year,omitempty"`
	Month       uint8    `msgpack:"month,omitempty"`
	Price       float64  `msgpack:"price,omitempty"`
	Currency    string   `msgpack:"currency,omitempty"`
	MarketValue float64  `msgpack:"market_value,omitempty"`
	Description string   `msgpack:"description,omitempty"`
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
	ID         int     `msgpack:"id,omitempty"`
	ItemUUID   string  `msgpack:"item_uuid,omitempty"`
	DatetimeMs int64   `msgpack:"date,omitempty"`
	Price      float64 `msgpack:"price,omitempty"`
	Unit       string  `msgpack:"unit,omitempty"`
	Currency   string  `msgpack:"currency,omitempty"`
}

type Packet struct {
	ID   int               `msgpack:"id,omitempty"`
	Type int16             `msgpack:"type,omitempty"`
	Meta map[string][]byte `msgpack:"meta,omitempty"`
	Body map[string][]byte `msgpack:"body,omitempty"`
}

func NewPacket(pkt *inventoryrpc.Packet) Packet {
	return Packet{
		ID:   pkt.ID,
		Type: pkt.Type,
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func ToInvPacket(pkt *Packet) inventoryrpc.Packet {
	return inventoryrpc.Packet{
		ID:   pkt.ID,
		Type: pkt.Type,
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func NewItem(item *inventory.Item) Item {
	return Item{
		ID:          item.ID,
		UUID:        item.UUID,
		Name:        item.Name,
		Description: item.Description,
		Unit:        item.Unit,
	}
}

func ToInvItem(item *Item) inventory.Item {
	return inventory.Item{
		ID:          item.ID,
		UUID:        item.UUID,
		Name:        item.Name,
		Description: item.Description,
		Unit:        item.Unit,
	}
}
