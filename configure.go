package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"github.com/hashicorp/hcl"
)

type Config struct {
	Hostname string `hcl: hostname`
}

type Config2 struct {
	//	Hostname string     `hcl: hostname`
	Hoge HogeConfig `hcl:"hoge"`
}

type HogeConfig struct {
	HogeDesc string     `hcl: "hogedesc"`
	Fuga     FugaConfig `hcl: "fuga"`
}

type FugaConfig struct {
	FugaDesc string `hcl: "fugadesc"`
}

func init() {
	d, err := ioutil.ReadFile("./config/config.hcl")
	if err != nil {
		log.Fatal(err)
	}

	//	var config Config
	//	err = hcl.Decode(&config, string(d))
	//	fmt.Println(config.Hostname)

	fmt.Println(string(d))
	var config2 Config2
	err = hcl.Decode(&config2, string(d))
	//	fmt.Println(config2.Hoge.HogeDesc)
	fmt.Println(err)
	fmt.Println(config2.Hoge)

}
