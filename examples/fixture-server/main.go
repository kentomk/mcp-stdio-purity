package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type message struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
}

func main() {
	mode := flag.String("mode", "clean", "clean, startup-banner, or late-log")
	flag.Parse()
	if *mode == "startup-banner" {
		fmt.Println("fixture startup banner")
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request message
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			continue
		}
		switch string(request.ID) {
		case "1":
			fmt.Println(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25","capabilities":{},"serverInfo":{"name":"fixture","version":"0"}}}`)
		case "2":
			fmt.Println(`{"jsonrpc":"2.0","id":2,"result":{}}`)
		default:
			if request.Method == "notifications/initialized" && *mode == "late-log" {
				fmt.Println("fixture late log")
			}
		}
	}
}
