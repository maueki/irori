
GOSRC=$(wildcard *.go)
COFFEEDIR=assets/js
COFFEESRC=$(wildcard $(COFFEEDIR)/*.coffee)

.PHONY: all coffee test
all: irori coffee

irori: $(GOSRC)
	go get -d -v ./... && go build -v .

coffee: $(COFFEESRC)
	coffee -o $(COFFEEDIR) -c $^

clean:
	go clean
	rm -f $(COFFEEDIR)/*.js

test:
	go test .
