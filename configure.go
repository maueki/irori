package main

import (
	"github.com/hashicorp/hcl"
	"io/ioutil"
	"log"
	"os"
)

var decoders []interface{}

func AddDecoder(decoder interface{}) { decoders = append(decoders, decoder) }

func ReadConfig() error {
	cp := os.Getenv("CONFIG_PATH")
	if cp == "" {
		cp = "./config/config.hcl"
	}

	s, err := ioutil.ReadFile(cp)
	if err != nil {
		log.Println(err)
		return err
	}

	readConfigStr(string(s))

	return nil
}

func readConfigStr(s string) {
	for _, dec := range decoders {
		hcl.Decode(dec, string(s))
	}
}
