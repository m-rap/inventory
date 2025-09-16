package inventoryrpc

import (
	"bytes"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

type Packet struct {
	H map[string][]byte `msgpack:"h,omitempty"`
	B map[string][]byte `msgpack:"b,omitempty"`
}

type PacketBuffer struct {
	buf bytes.Buffer
}

func (pb *PacketBuffer) Feed(data []byte) ([]*Packet, error) {
	pb.buf.Write(data)

	var results []*Packet
	dec := msgpack.NewDecoder(&pb.buf)

	for {
		v := new(Packet)
		if err := dec.Decode(&v); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// not enough data yet, stop
				break
			}
			return results, err
		}
		results = append(results, v)
	}
	return results, nil
}
