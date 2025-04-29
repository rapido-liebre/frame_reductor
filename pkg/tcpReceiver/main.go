package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
)

func main() {
	port := flag.Int("port", 7420, "Port TCP do nasłuchu (domyślnie 7420)")
	flag.Parse()

	address := fmt.Sprintf(":%d", *port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[Receiver] Błąd nasłuchu: %v", err)
	}
	defer listener.Close()
	log.Printf("[Receiver] Nasłuch na %s", address)

	conn, err := listener.Accept()
	if err != nil {
		log.Fatalf("[Receiver] Błąd akceptacji połączenia: %v", err)
	}
	defer conn.Close()
	log.Printf("[Receiver] Połączono z %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		log.Printf("[Receiver] Otrzymano: %s", text)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[Receiver] Błąd odczytu: %v", err)
	}
}
