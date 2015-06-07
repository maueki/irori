
GOSRCDIR=./src
COFFEEDIR=assets/js
COFFEESRC=$(wildcard $(COFFEEDIR)/*.coffee)


ifeq ($(OS),Windows_NT)
IRORI_BINNAME=irori.exe
else
IRORI_BINNAME=irori
endif

.PHONY: all coffee test
all: irori coffee

irori:
	go get -d -v $(GOSRCDIR)
	go build -v -o $(IRORI_BINNAME) $(GOSRCDIR)

coffee: $(COFFEESRC)
	coffee -o $(COFFEEDIR) -c $^

clean:
	rm $(IRORI_BINNAME)
	go clean
	rm -f $(COFFEEDIR)/*.js

test:
	go test $(GOSRCDIR)
