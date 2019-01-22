package main

import (
	"encoding/json"
	"log"
	"os"
	"./messages"
)

func main() {
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	configuration := messages.Config{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Panic(err)
	}
	messages.Send(configuration)
}
