
GOSRCDIR=./src
GOSRCS=$(wildcard $(GOSRCDIR)/*.go)
COFFEEDIR=assets/js
COFFEESRC=$(wildcard $(COFFEEDIR)/*.coffee)
LBJS=assets/js/lib.js

ifeq ($(OS),Windows_NT)
IRORI_BINNAME=irori.exe
else
IRORI_BINNAME=irori
endif

.PHONY: all coffee test libjs
all: irori coffee

libjs: bower.json
	gulp create-libjs

irori: $(GOSRCS) libjs
	go get -d -v $(GOSRCDIR)
	go build -v -o $(IRORI_BINNAME) $(GOSRCDIR)

coffee: $(COFFEESRC)
	coffee -o $(COFFEEDIR) -c $^

clean:
	rm $(IRORI_BINNAME)
	go clean
	rm -f $(COFFEEDIR)/*.js
	rm -f $(LIBJS)
test:
	go test $(GOSRCDIR)
