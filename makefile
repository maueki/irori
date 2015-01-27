
GOSRC=main.go filter.go
COFFEEDIR=assets/js
COFFEESRC=$(COFFEEDIR)/editor.coffee

.PHONY: all
all: go_wiki coffee

go_wiki: $(GOSRC)
	go get -d -v ./... && go build -v

coffee: $(COFFEESRC)
	cd $(COFFEEDIR) && coffee -c *.coffee

.PHONY: coffee
