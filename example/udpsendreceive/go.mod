module inventory-udpsendreceive-example

go 1.23

toolchain go1.24.7

replace inventory => ../..

replace inventoryrpc => ../../rpc

replace inventorypb => ../../pb

replace inventorymsgpack => ../../msgpack

replace inventoryexamplecommon => ../common

require inventory v0.0.0-00010101000000-000000000000 // indirect

require (
	inventoryexamplecommon v0.0.0-00010101000000-000000000000
	inventoryrpc v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	inventorymsgpack v0.0.0-00010101000000-000000000000 // indirect
	inventorypb v0.0.0-00010101000000-000000000000 // indirect
)
