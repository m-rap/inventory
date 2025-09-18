#!/bin/bash

if [ ! -f gen.sh ]; then
    echo "wrong dir. aborting."
    exit 1
fi

protoc -I . --go_out=. inventory.proto 
mv inventory/inventorypb/inventory.pb.go .
rm -r inventory