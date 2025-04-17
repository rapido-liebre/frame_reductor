package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Definicja flag
	port := flag.Int("port", 12345, "Port number to listen on")
	timeout := flag.Int("timeout", 60, "Timeout in seconds for the server to stop listening")
	mode := flag.String("mode", "client", "TCP mode: client (default) or server")

	flag.Parse()

	if *mode == "" || *mode != "server" && *mode != "client" {
		fmt.Println("Invalid TCP mode. Use client or server.")
		os.Exit(1)
	}

	switch *mode {
	case "client":
		StartTCPClient(*port)
	case "server":
		StartTCPServer(*port, *timeout)
	}
}
