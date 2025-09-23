module inventoryexamplecommon

go 1.23

toolchain go1.24.7

replace inventory => ../..

replace inventoryrpc => ../../rpc

replace inventorypb => ../../pb

replace inventorymsgpack => ../../msgpack

require (
	github.com/google/uuid v1.6.0
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	google.golang.org/protobuf v1.36.9
	inventory v0.0.0-00010101000000-000000000000
	inventorypb v0.0.0-00010101000000-000000000000
	inventoryrpc v0.0.0-00010101000000-000000000000
	inventorymsgpack v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
