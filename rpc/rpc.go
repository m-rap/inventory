package inventoryrpc

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"

	"github.com/google/uuid"
)

var (
	PacketMagic = []byte{0xa0, 0xa1}
)

const (
	TypeReq  = 0
	TypeResp = 1
)

type PacketWrapper struct {
	Length      int
	PacketBytes []byte
	Checksum    uint32
}

type Packet struct {
	UUID uuid.UUID
	Type int16
	Meta map[string][]byte
	Body map[string][]byte
}

func EncodePacketWrapper(packet []byte) ([]byte, error) {
	pkt := PacketWrapper{
		PacketBytes: packet,
	}

	// msgPackData := PacketMsgPack{}
	// msgPackData.H2.ID = id
	// msgPackData.H2.Type = TypeReq
	// msgPackData.H2.Meta = meta
	// msgPackData.Body = body

	// msgPackBin, err := msgpack.Marshal(&msgPackData)
	// if err != nil {
	// 	return nil, err
	// }

	tmpB4 := make([]byte, 4)

	buf := bytes.Buffer{}
	buf.Write(PacketMagic) // magic
	binary.LittleEndian.PutUint32(tmpB4, uint32(2+4+len(pkt.PacketBytes)+4))
	buf.Write(tmpB4) // length
	buf.Write(pkt.PacketBytes)
	data := buf.Bytes()
	crc := crc32.ChecksumIEEE(data)
	binary.LittleEndian.PutUint32(tmpB4, crc)
	buf.Write(tmpB4) // checksum

	return buf.Bytes(), nil
}

type PacketBuffer struct {
	buf bytes.Buffer
}

func (pb *PacketBuffer) Feed(data []byte) ([]*PacketWrapper, error) {
	pb.buf.Write(data)

	var results []*PacketWrapper
	// dec := msgpack.NewDecoder(&pb.buf)

	for {
		if pb.buf.Len() < 2+4+4 {
			break
		}

		pkt := new(PacketWrapper)
		b := pb.buf.Bytes()

		if b[0] != PacketMagic[0] || b[1] != PacketMagic[1] {
			// invalid magic, skip byte
			pb.buf.Next(1)
			continue
		}

		length := int(binary.LittleEndian.Uint32(b[2:6]))
		if length < 2+4+4 {
			// invalid length, skip magic
			pb.buf.Next(2)
			continue
		}

		if pb.buf.Len() < length {
			break
		}

		checksum := binary.LittleEndian.Uint32(b[length-4 : length])
		crc := crc32.ChecksumIEEE(b[:length-4])
		if crc != checksum {
			// invalid checksum, skip this packet
			pb.buf.Next(int(length))
			continue
		}

		packetBytes := b[6 : length-4]
		// err := msgpack.NewDecoder(bytes.NewReader(msgPackData)).Decode(&pkt.Body)
		// if err != nil {
		// 	pb.buf.Next(int(length))
		// 	return results, err
		// }

		pkt.PacketBytes = packetBytes
		pkt.Checksum = checksum
		pkt.Length = int(length)

		pb.buf.Next(int(length))

		results = append(results, pkt)
	}
	return results, nil
}
