
GOSRC=main.go filter.go db/db.go
COFFEEDIR=assets/js
COFFEESRC=$(wildcard $(COFFEEDIR)/*.coffee)

.PHONY: all coffee test
all: go_wiki coffee

go_wiki: $(GOSRC)
	go get -d -v ./... && go build -v ./...

coffee: $(COFFEESRC)
	coffee -o $(COFFEEDIR) -c $^

test:
	go test ./...
