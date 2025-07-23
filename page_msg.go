package main

import (
	"io"
	"log"
	"os"
)

var MessagePage string

func getMessagePage() string {
	if len(MessagePage) > 0 {
		return MessagePage
	}
	file, err := os.Open("t.html")
	if err != nil {
		log.Println("open t error:", err)
		return ""
	}
	defer file.Close()
	bt, _ := io.ReadAll(file)
	MessagePage = string(bt)
	return MessagePage
}
