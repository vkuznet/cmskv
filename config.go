package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// Configuration stores server configuration parameters
type Configuration struct {
	Port          int    `json:"port"`     // server port number
	Base          string `json:"base"`     // base URL
	Verbose       int    `json:"verbose"`  // verbose output
	UTC           bool   `json:"utc"`      // report logger time in UTC
	BadgerDB      string `json:"db"`       // db file name
	LimiterPeriod string `json:"rate"`     // github.com/ulule/limiter rate value
	LogFile       string `json:"log_file"` // server log file
	SHA           string `json:"sha"`      // sha version: sha1, sha256, sha512
}

// Config variable represents configuration object
var Config Configuration

// String returns string representation of dbs Config
func (c *Configuration) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		log.Println("ERROR: fail to marshal configuration", err)
		return ""
	}
	return string(data)
}

// helper function to parse configuration
func parseConfig(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Println("Unable to read", err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		log.Println("Unable to parse", err)
		return err
	}
	if Config.Port == 0 {
		Config.Port = 9212
	}
	if Config.BadgerDB == "" {
		Config.BadgerDB = "/tmp/badger.db"
	}
	if Config.LimiterPeriod == "" {
		Config.LimiterPeriod = "100-S"
	}
	return nil
}
