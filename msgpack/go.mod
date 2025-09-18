module inventorymsgpack

go 1.22.3

replace inventory => ..

replace inventoryrpc => ../rpc

require inventory v0.0.0-00010101000000-000000000000

require inventoryrpc v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
)
