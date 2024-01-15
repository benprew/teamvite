package teamvite

import (
	"encoding/json"
	"log"
	"os"
)

var CONFIG Config
var DefaultConfigPath = "config.json"

type Config struct {
	Servername string `json:"servername"` // teamvite.com, teamvitedev.com
	SMTP       SMTPConfig
	SMS        SMSConfig
}

type SMTPConfig struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type SMSConfig struct {
	Sid   string `json:"sid"`
	Token string `json:"token"`
	API   string `json:"api"`
	From  string `json:"from"` // From phone number
}

func LoadConfig(filename string) (c Config, err error) {
	f, err := os.ReadFile(filename)
	if err != nil {
		return
		// log.Fatalln("[FATAL]: config file not found: ", configFile)
	}
	err = json.Unmarshal(f, &c)
	return
}

func DefaultConfig() (c Config) {
	c, err := LoadConfig(DefaultConfigPath)
	if err != nil {
		log.Fatalln("[FAIL]: Unable to load config ", DefaultConfigPath, err)
	}
	return c
}
