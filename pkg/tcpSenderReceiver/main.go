package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

func main() {
	port := flag.Int("port", 7420, "TCP port do wysyłania i odbioru ramek (domyślnie 7420)")
	flag.Parse()

	address := fmt.Sprintf("localhost:%d", *port)

	go startSender(address) // Najpierw wystartuj serwer wysyłający dane

	time.Sleep(5 * time.Second) // Poczekaj kilka sekund

	go startReceiver(address) // Uruchom odbiorcę

	// Zatrzymaj główną funkcję
	select {}
}

// Sender - serwer TCP, wysyła losowe teksty
func startSender(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[Sender] Błąd startu serwera: %v", err)
	}
	defer listener.Close()
	log.Printf("[Sender] Nasłuch na %s", address)

	conn, err := listener.Accept()
	if err != nil {
		log.Fatalf("[Sender] Błąd akceptacji połączenia: %v", err)
	}
	defer conn.Close()
	log.Printf("[Sender] Połączono z %s", conn.RemoteAddr())

	writer := bufio.NewWriter(conn)
	for {
		text := randomText()
		log.Printf("[Sender] Wysyłam: %s", text)
		_, err := writer.WriteString(text + "\n")
		if err != nil {
			log.Printf("[Sender] Błąd wysyłki: %v", err)
			return
		}
		writer.Flush()

		time.Sleep(2 * time.Second)
	}
}

// Receiver - klient TCP, odbiera teksty
func startReceiver(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalf("[Receiver] Błąd połączenia: %v", err)
	}
	defer conn.Close()
	log.Printf("[Receiver] Połączono z %s", address)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		log.Printf("[Receiver] Otrzymano: %s", text)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[Receiver] Błąd odczytu: %v", err)
	}
}

// Funkcja do generowania losowego tekstu
func randomText() string {
	words := []string{
		"Alpha", "Bravo", "Charlie", "Delta", "Echo",
		"Foxtrot", "Golf", "Hotel", "India", "Juliet",
	}
	rand.Seed(time.Now().UnixNano())
	return words[rand.Intn(len(words))]
}
