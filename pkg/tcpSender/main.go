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
	port := flag.Int("port", 7420, "Port TCP do wysłania danych (domyślnie 7420)")
	flag.Parse()

	address := fmt.Sprintf("localhost:%d", *port)

	// Połącz się jako klient TCP
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalf("[Sender] Błąd połączenia: %v", err)
	}
	defer conn.Close()
	log.Printf("[Sender] Połączono z %s", address)

	writer := bufio.NewWriter(conn)

	for {
		text := randomText()
		log.Printf("[Sender] Wysyłam: %s", text)

		_, err := writer.WriteString(text + "\n")
		if err != nil {
			log.Printf("[Sender] Błąd wysyłki: %v", err)
			break
		}
		writer.Flush()

		time.Sleep(2 * time.Second)
	}
}

// Funkcja generująca losowy tekst
func randomText() string {
	words := []string{
		"Alpha", "Bravo", "Charlie", "Delta", "Echo",
		"Foxtrot", "Golf", "Hotel", "India", "Juliet",
	}
	rand.Seed(time.Now().UnixNano())
	return words[rand.Intn(len(words))]
}
