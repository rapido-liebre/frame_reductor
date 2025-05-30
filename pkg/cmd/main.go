package main

import (
	"flag"
	"fmt"
	"frame_reductor/handler"
	"frame_reductor/model"
	"log"
	"net"
	"strconv"
	"strings"
	//"time"
)

func main() {
	// Definicja flag
	mode := flag.String("mode", "listen", "Mode of operation: listen (default) or file")
	tcpMode := flag.String("tcp_mode", "client", "TCP mode: client (default) or server")
	ports := flag.String("ports", "4716", "Comma-separated list of UDP ports to listen on, e.g., 4716,4720,5002")
	timeout := flag.Int("time", 0, "Timeout in seconds (used only in 'listen' mode)")
	frames := flag.Int("frames", 10, "Number of frames: 1, 2, 5, 10, 20, 25, 50")
	outputPort := flag.String("output_port", "", "Output protocol and port in format [TCP|UDP]:<port> (e.g., UDP:7420 or TCP:7421)")
	targetHost := flag.String("target_host", "localhost", "Target host for TCP client mode (e.g., 192.168.1.10)")
	bindIP := flag.String("bind", "", "Local IP address (e.g. 192.168.1.100) through which the connection is to be established")
	inputFile := flag.String("input_file", "", "Path to the input file from which the data will be loaded")
	outputFile := flag.String("output_file", "", "Path to the output file where the data will be saved")
	showInterfaces := flag.Bool("show_interfaces", false, "Show interfaces")
	checkTcpConnection := flag.Bool("check_tcp_connection", false, "Check TCP connection in client mode")

	// Parsowanie flag
	flag.Parse()

	if *showInterfaces {
		fmt.Println("Dostępne interfejsy:")
		ifaces, _ := net.Interfaces()
		for _, iface := range ifaces {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				fmt.Printf("Interfejs %s: %v\n", iface.Name, addr)
			}
		}
		return
	}

	// Walidacja wartości flag
	if *mode != "listen" && *mode != "file" {
		log.Fatalf("Invalid mode. Use 'listen' or 'file'.")
	}

	validFrames := map[int]bool{1: true, 2: true, 4: true, 5: true, 10: true, 20: true, 25: true, 40: true, 50: true}
	if !validFrames[*frames] {
		log.Fatalf("Invalid value for 'frames'. Allowed values: 1, 2, 4, 5, 10, 20, 25, 40, 50.")
	}
	model.OutputDataRate = float64(*frames)

	if *outputPort != "" {
		parts := strings.Split(*outputPort, ":")
		if len(parts) != 2 {
			log.Fatalf("Invalid output_port format. Use [TCP|UDP]:<port>.")
		}

		protocol := strings.ToUpper(parts[0])
		if protocol != "TCP" && protocol != "UDP" {
			log.Fatalf("Invalid protocol in output_port. Use TCP or UDP.")
		}

		outPort, err := strconv.Atoi(parts[1])
		if err != nil || outPort < 1 || outPort > 65535 {
			log.Fatalf("Invalid port in output_port. Must be a valid integer between 1 and 65535.")
		}
		fmt.Printf("Output protocol: %s, Port: %d\n", protocol, outPort)
		model.Out.Protocol = model.Protocol(protocol)
		model.Out.Port = uint32(outPort)
	}

	if model.Out.Protocol == model.ProtocolTCP {
		if *tcpMode == "" || *tcpMode != "server" && *tcpMode != "client" {
			log.Fatalf("Invalid TCP mode. Use client or server.")
		}
		model.Out.TCPMode = model.TCPMode(*tcpMode)
	}

	var portList []int
	for _, p := range strings.Split(*ports, ",") {
		port, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || port < 1 || port > 65535 {
			log.Fatalf("Nieprawidłowy port: %s", p)
		}
		portList = append(portList, port)
	}

	fmt.Printf("Porty UDP do nasłuchu: %v\n", portList)

	model.Out.TargetHost = *targetHost
	model.Out.BindIP = *bindIP

	if *checkTcpConnection {
		if err := model.Out.CanConnectAsTCPClient(); err != nil {
			log.Fatalf("TCP connection as client mode cannot be established. Error: %v\n.", err.Error())
		}
		handler.CheckTCPClientConnection(model.Out.Port, model.Out.TargetHost, model.Out.BindIP)
		return
	}

	var frameChan chan []byte

	if model.Out.Protocol == model.ProtocolTCP {
		frameChan = make(chan []byte)
	}

	if model.Out.Protocol == model.ProtocolTCP {
		switch model.Out.TCPMode {
		case model.TCPServer:
			//go handler.StartTCPServer(*ports, frameChan) //TODO
		case model.TCPClient:
			go handler.StartTCPClient(model.Out.Port, model.Out.TargetHost, model.Out.BindIP, frameChan)
		}
	}

	// Obsługa trybu działania
	switch *mode {
	case "listen":
		fmt.Printf("Starting in 'listen' mode on ports %s with timeout %d seconds and frames %d...\n", *ports, *timeout, *frames)
		for _, p := range portList {
			go handler.StartListening(p, *timeout, *outputFile, frameChan)
		}
	case "file":
		fmt.Printf("Starting in 'file' mode with data rate %d frames/sec...\n", *frames)
		handler.ProcessFile(frameChan, *inputFile)
	}
}
