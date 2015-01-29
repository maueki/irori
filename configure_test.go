package main

import (
	"testing"
)

var configStr string = `
hostname = "test host name"

smtp_settings = {
    address = "smtp.example.com"
    port = 587
    user_name = "username@example.com"
    password = "pass"
}
`

// member name and hcl name must be same (ignore case)
type MainConfig struct {
	HostName string
}

type Smtp struct {
	Smtp_Settings SmtpSettings
}

type SmtpSettings struct {
	Address   string
	Port      int
	User_Name string
	Password  string
}

var mainConfig MainConfig
var smtpSettings Smtp

func TestConfigure(t *testing.T) {

	AddDecoder(&mainConfig)
	AddDecoder(&smtpSettings)

	readConfigStr(configStr)

	if mainConfig.HostName != "test host name" {
		t.Error("unexpected", mainConfig.HostName)
	}

	s := &smtpSettings.Smtp_Settings
	if s.Address != "smtp.example.com" {
		t.Error("unexpected", s.Address)
	}

	if s.Port != 587 {
		t.Error("unexpected port", s.Address)
	}

	if s.User_Name != "username@example.com" {
		t.Error("unexpected user_name", s.User_Name)
	}

	if s.Password != "pass" {
		t.Error("unexpected pass", s.User_Name)
	}
}
