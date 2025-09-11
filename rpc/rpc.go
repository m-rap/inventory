package inventoryrpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"inventory"
	"math"
	"time"
)

// ---------------- Encoder/Decoder ----------------

type Encoder struct {
	buf bytes.Buffer
}

type Decoder struct {
	r *bytes.Reader
}

func NewEncoder() *Encoder            { return &Encoder{} }
func NewDecoder(data []byte) *Decoder { return &Decoder{r: bytes.NewReader(data)} }

func (e *Encoder) Bytes() []byte { return e.buf.Bytes() }

// Flexible length encoding (Bitcoin-style):
// 0..252 = 1 byte
// 0xFD + 2-byte
// 0xFE + 4-byte
// 0xFF + 8-byte
func (e *Encoder) WriteLength(n int) {
	switch {
	case n <= 252:
		e.buf.WriteByte(byte(n))
	case n <= math.MaxUint16:
		e.buf.WriteByte(0xFD)
		binary.Write(&e.buf, binary.LittleEndian, uint16(n))
	case n <= math.MaxUint32:
		e.buf.WriteByte(0xFE)
		binary.Write(&e.buf, binary.LittleEndian, uint32(n))
	default:
		e.buf.WriteByte(0xFF)
		binary.Write(&e.buf, binary.LittleEndian, uint64(n))
	}
}

func (d *Decoder) ReadLength() (int, error) {
	prefix, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	switch prefix {
	case 0xFF:
		var v uint64
		if err := binary.Read(d.r, binary.LittleEndian, &v); err != nil {
			return 0, err
		}
		return int(v), nil
	case 0xFE:
		var v uint32
		if err := binary.Read(d.r, binary.LittleEndian, &v); err != nil {
			return 0, err
		}
		return int(v), nil
	case 0xFD:
		var v uint16
		if err := binary.Read(d.r, binary.LittleEndian, &v); err != nil {
			return 0, err
		}
		return int(v), nil
	default:
		return int(prefix), nil
	}
}

func (e *Encoder) WriteString(s string) {
	e.WriteLength(len(s))
	e.buf.WriteString(s)
}

func (d *Decoder) ReadString() (string, error) {
	n, err := d.ReadLength()
	if err != nil {
		return "", err
	}
	buf := make([]byte, n)
	if _, err := d.r.Read(buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (e *Encoder) WriteFloat64(v float64) {
	binary.Write(&e.buf, binary.LittleEndian, v)
}

func (d *Decoder) ReadFloat64() (float64, error) {
	var v float64
	err := binary.Read(d.r, binary.LittleEndian, &v)
	return v, err
}

func (e *Encoder) WriteTime(t time.Time) {
	binary.Write(&e.buf, binary.LittleEndian, t.UnixNano())
}

func (d *Decoder) ReadTime() (time.Time, error) {
	var ns int64
	if err := binary.Read(d.r, binary.LittleEndian, &ns); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, ns), nil
}

// ---------------- High-level encode/decode ----------------

func (e *Encoder) WriteItem(it inventory.Item) {
	e.WriteString(it.ID)
	e.WriteString(it.Name)
	e.WriteString(it.Description)
	e.WriteString(it.Unit)
}

func (d *Decoder) ReadItem() (inventory.Item, error) {
	id, err := d.ReadString()
	if err != nil {
		return inventory.Item{}, err
	}
	name, err := d.ReadString()
	if err != nil {
		return inventory.Item{}, err
	}
	desc, err := d.ReadString()
	if err != nil {
		return inventory.Item{}, err
	}
	unit, err := d.ReadString()
	if err != nil {
		return inventory.Item{}, err
	}
	return inventory.Item{ID: id, Name: name, Description: desc, Unit: unit}, nil
}

func (e *Encoder) WriteTransactionLine(l inventory.TransactionLine) {
	e.WriteString(l.TransactionID)
	e.WriteString(l.AccountID)
	e.WriteString(l.ItemID)
	e.WriteFloat64(l.Quantity)
	e.WriteString(l.Unit)
	e.WriteFloat64(l.Price)
	e.WriteString(l.Currency)
	e.WriteString(l.Note)
}

func (d *Decoder) ReadTransactionLine() (inventory.TransactionLine, error) {
	trID, err := d.ReadString()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	accID, err := d.ReadString()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	itemID, err := d.ReadString()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	qty, err := d.ReadFloat64()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	unit, err := d.ReadString()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	price, err := d.ReadFloat64()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	currency, err := d.ReadString()
	if err != nil {
		return inventory.TransactionLine{}, err
	}
	note, err := d.ReadString()
	if err != nil {
		return inventory.TransactionLine{}, err
	}

	return inventory.TransactionLine{
		TransactionID: trID, AccountID: accID, ItemID: itemID,
		Quantity: qty, Unit: unit, Price: price, Currency: currency, Note: note,
	}, nil
}

func (e *Encoder) WriteTransaction(t inventory.Transaction) {
	e.WriteString(t.ID)
	e.WriteString(t.Description)
	e.WriteTime(t.Date)
	e.WriteLength(len(t.Lines))
	for _, line := range t.Lines {
		e.WriteTransactionLine(line)
	}
}

func (d *Decoder) ReadTransaction() (inventory.Transaction, error) {
	id, err := d.ReadString()
	if err != nil {
		return inventory.Transaction{}, err
	}
	desc, err := d.ReadString()
	if err != nil {
		return inventory.Transaction{}, err
	}
	date, err := d.ReadTime()
	if err != nil {
		return inventory.Transaction{}, err
	}
	n, err := d.ReadLength()
	if err != nil {
		return inventory.Transaction{}, err
	}

	lines := make([]inventory.TransactionLine, 0, n)
	for i := 0; i < n; i++ {
		line, err := d.ReadTransactionLine()
		if err != nil {
			return inventory.Transaction{}, err
		}
		lines = append(lines, line)
	}
	return inventory.Transaction{ID: id, Description: desc, Date: date, Lines: lines}, nil
}

func (e *Encoder) WriteBalance(b inventory.Balance) {
	e.WriteLength(len(b.Path))
	for _, p := range b.Path {
		e.WriteString(p)
	}
	e.WriteString(b.AccountID)
	e.WriteString(b.Item)
	e.WriteString(b.Unit)
	e.WriteFloat64(b.Quantity)
	e.WriteFloat64(b.AvgCost)
	e.WriteFloat64(b.Value)
	e.WriteTime(b.Date)
	e.WriteFloat64(b.Price)
	e.WriteString(b.Currency)
	e.WriteFloat64(b.MarketValue)
	e.WriteString(b.Description)
}

func (d *Decoder) ReadBalance() (inventory.Balance, error) {
	n, err := d.ReadLength()
	if err != nil {
		return inventory.Balance{}, err
	}
	path := make([]string, 0, n)
	for i := 0; i < n; i++ {
		s, err := d.ReadString()
		if err != nil {
			return inventory.Balance{}, err
		}
		path = append(path, s)
	}
	accountID, err := d.ReadString()
	if err != nil {
		return inventory.Balance{}, err
	}
	item, err := d.ReadString()
	if err != nil {
		return inventory.Balance{}, err
	}
	unit, err := d.ReadString()
	if err != nil {
		return inventory.Balance{}, err
	}
	qty, err := d.ReadFloat64()
	if err != nil {
		return inventory.Balance{}, err
	}
	avgCost, err := d.ReadFloat64()
	if err != nil {
		return inventory.Balance{}, err
	}
	value, err := d.ReadFloat64()
	if err != nil {
		return inventory.Balance{}, err
	}
	date, err := d.ReadTime()
	if err != nil {
		return inventory.Balance{}, err
	}
	price, err := d.ReadFloat64()
	if err != nil {
		return inventory.Balance{}, err
	}
	currency, err := d.ReadString()
	if err != nil {
		return inventory.Balance{}, err
	}
	mv, err := d.ReadFloat64()
	if err != nil {
		return inventory.Balance{}, err
	}
	desc, err := d.ReadString()
	if err != nil {
		return inventory.Balance{}, err
	}

	return inventory.Balance{
		Path: path, AccountID: accountID, Item: item, Unit: unit,
		Quantity: qty, AvgCost: avgCost, Value: value, Date: date,
		Price: price, Currency: currency, MarketValue: mv, Description: desc,
	}, nil
}

// ---------------- Message Wrapper ----------------

type Message struct {
	Type byte
	Data []byte
}

func EncodeMessage(msgType byte, inner func(enc *Encoder)) []byte {
	e := NewEncoder()
	e.buf.WriteByte(msgType)
	inner(e)
	return e.Bytes()
}

func DecodeMessage(data []byte) (byte, *Decoder, error) {
	if len(data) < 1 {
		return 0, nil, errors.New("empty message")
	}
	msgType := data[0]
	return msgType, NewDecoder(data[1:]), nil
}
