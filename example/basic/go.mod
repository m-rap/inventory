module inventory-basic-example

go 1.22.3

replace inventory => ../..

require inventory v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
)
