module msgpackprotobufcompare

go 1.23

toolchain go1.24.7

replace inventory => ../..

replace inventoryrpc => ../../rpc

replace inventorypb => ../../pb

replace inventorymsgpack => ../../msgpack

require inventory v0.0.0-00010101000000-000000000000

require (
	github.com/vmihailenco/msgpack/v5 v5.4.1
	google.golang.org/protobuf v1.36.9
	inventorymsgpack v0.0.0-00010101000000-000000000000
	inventorypb v0.0.0-00010101000000-000000000000
	inventoryrpc v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
)
