#!/bin/bash -ex

if [ -z $GOPATH ]; then
    echo "please set \$GOPATH"
    exit 1
fi

go get github.com/coopernurse/gorp
go get github.com/flosch/pongo2
go get github.com/flosch/pongo2-addons
go get github.com/gorilla/sessions
go get github.com/mattn/go-sqlite3
go get github.com/zenazn/goji
go get github.com/flosch/pongo2-addons
