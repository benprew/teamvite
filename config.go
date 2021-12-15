package main

import (
	"encoding/json"
	"log"
	"os"
)

var CONFIG = readConfig()

type config struct {
	Servername string `json:"servername"` // teamvite.com, teamvitedev.com
	SMTP       SMTPConfig
}

type SMTPConfig struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func readConfig() (c config) {
	configFile := "config.json"
	f, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalln("[FATAL]: config file not found: ", configFile)
	}
	if err = json.Unmarshal(f, &c); err != nil {
		log.Fatalln("[FATAL]: unable to parse config file:", err)
	}
	return

}
