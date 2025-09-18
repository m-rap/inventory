package inventorypb

import (
	"inventory"
	"inventoryrpc"
)

func ToInvPacket(pkt *Packet) inventoryrpc.Packet {
	return inventoryrpc.Packet{
		ID:   int(pkt.ID),
		Type: int16(pkt.Type),
		Meta: pkt.Meta,
		Body: pkt.Body,
	}
}

func NewItem(item *inventory.Item) Item {
	return Item{
		ID:          int32(item.ID),
		UUID:        item.UUID,
		Name:        item.Name,
		Description: item.Description,
		Unit:        item.Unit,
	}
}

func ToInvItem(item *Item) inventory.Item {
	return inventory.Item{
		ID:          int(item.ID),
		UUID:        item.UUID,
		Name:        item.Name,
		Description: item.Description,
		Unit:        item.Unit,
	}
}
