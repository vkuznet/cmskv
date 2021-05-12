package main

// cmskv - CMS persistent key-value store
//
// Copyright (c) 2021 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"flag"
	"log"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

func main() {
	var config string
	flag.StringVar(&config, "config", "config.json", "server config file")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "Verbose level, support 0,1,2")
	flag.Parse()
	err := parseConfig(config)
	if err != nil {
		log.Fatalf("Unable to parse config file %s, error: %v", config, err)
	}
	log.SetFlags(0)
	if Config.Verbose > 0 {
		log.SetFlags(log.Lshortfile)
	}
	log.SetOutput(new(logWriter))
	if Config.LogFile != "" {
		rl, err := rotatelogs.New(Config.LogFile + "-%Y%m%d")
		if err == nil {
			rotlogs := rotateLogWriter{RotateLogs: rl}
			log.SetOutput(rotlogs)
		}
	}
	if err != nil {
		log.Printf("Unable to parse, time: %v, config: %v\n", time.Now(), config)
	}
	log.Println("Configuration:", Config.String())
	server()
}
