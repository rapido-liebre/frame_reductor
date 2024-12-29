package main

import (
	"flag"
	"fmt"
	"frame_reductor/handler"
	//"net"
	"os"
	//"time"
)

func main() {
	// Definicja flag
	mode := flag.String("mode", "listen", "Mode of operation: listen or file")
	port := flag.Int("port", 4716, "Port number to listen on (used only in 'listen' mode)")
	timeout := flag.Int("time", 60, "Timeout in seconds (used only in 'listen' mode)")

	// Parsowanie flag
	flag.Parse()

	// Walidacja wartości flag
	if *mode != "listen" && *mode != "file" {
		fmt.Println("Invalid mode. Use 'listen' or 'file'.")
		os.Exit(1)
	}

	// Obsługa trybu działania
	switch *mode {
	case "listen":
		fmt.Printf("Starting in 'listen' mode on port %d with timeout %d seconds...\n", *port, *timeout)
		handler.StartListening(*port, *timeout)
	case "file":
		fmt.Println("Starting in 'file' mode...")
		handler.ProcessFile()
	}
}
