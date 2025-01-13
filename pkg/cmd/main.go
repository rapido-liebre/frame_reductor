package main

import (
	"flag"
	"fmt"
	"frame_reductor/handler"
	"frame_reductor/model"
	"strconv"
	"strings"

	//"net"
	"os"
	//"time"
)

func main() {
	// Definicja flag
	mode := flag.String("mode", "listen", "Mode of operation: listen or file")
	port := flag.Int("port", 4716, "Port number to listen on (used only in 'listen' mode)")
	timeout := flag.Int("time", 60, "Timeout in seconds (used only in 'listen' mode)")
	frames := flag.Int("frames", 10, "Number of frames: 1, 2, 5, 10, 20, 25, 50")
	outputPort := flag.String("output_port", "", "Output protocol and port in format [TCP|UDP]:<port>, e.g., UDP:7420 or TCP:7421")

	// Parsowanie flag
	flag.Parse()

	// Walidacja wartości flag
	if *mode != "listen" && *mode != "file" {
		fmt.Println("Invalid mode. Use 'listen' or 'file'.")
		os.Exit(1)
	}

	validFrames := map[int]bool{1: true, 2: true, 4: true, 5: true, 10: true, 20: true, 25: true, 40: true, 50: true}
	if !validFrames[*frames] {
		fmt.Println("Invalid value for 'frames'. Allowed values: 1, 2, 4, 5, 10, 20, 25, 40, 50.")
		os.Exit(1)
	}
	model.FramesCount = uint32(*frames)

	if *outputPort != "" {
		parts := strings.Split(*outputPort, ":")
		if len(parts) != 2 {
			fmt.Println("Invalid output_port format. Use [TCP|UDP]:<port>.")
			os.Exit(1)
		}

		protocol := strings.ToUpper(parts[0])
		if protocol != "TCP" && protocol != "UDP" {
			fmt.Println("Invalid protocol in output_port. Use TCP or UDP.")
			os.Exit(1)
		}

		outPort, err := strconv.Atoi(parts[1])
		if err != nil || outPort < 1 || outPort > 65535 {
			fmt.Println("Invalid port in output_port. Must be a valid integer between 1 and 65535.")
			os.Exit(1)
		}
		fmt.Printf("Output protocol: %s, Port: %s\n", protocol, outPort)
		model.Out.Protocol = model.Protocol(protocol)
		model.Out.Port = uint32(outPort)
	}

	// Obsługa trybu działania
	switch *mode {
	case "listen":
		fmt.Printf("Starting in 'listen' mode on port %d with timeout %d seconds and frames %d...\n", *port, *timeout, *frames)
		handler.StartListening(*port, *timeout)
	case "file":
		fmt.Printf("Starting in 'file' mode with frames %d...\n", *frames)
		handler.ProcessFile()
	}
}
