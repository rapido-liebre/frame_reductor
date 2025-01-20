package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	// Bajty do wysłania
	sentBytes := []byte{170, 1, 0, 20, 184, 110, 103, 118, 237, 182, 0, 0, 1, 44, 0, 128, 66, 71, 248, 100} //66, 71, 248, 0}

	// Uruchom serwer w goroutine
	go startTCPServer(7420)

	// Poczekaj, aż serwer się uruchomi
	time.Sleep(20 * time.Second)

	// Wyślij dane do serwera
	sendTCPData("localhost:7420", sentBytes)
}

func startTCPServer(port int) {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Błąd podczas uruchamiania serwera: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Serwer nasłuchuje na porcie %d...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Błąd podczas akceptowania połączenia: %v\n", err)
			continue
		}

		go handleConnection(conn, 20)
	}
}

func handleConnection(conn net.Conn, dataLength int) {
	defer conn.Close()

	// Odczytaj dane z połączenia
	buffer := make([]byte, dataLength)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Błąd podczas odczytu danych: %v\n", err)
		return
	}

	// Wyświetl odebrane bajty
	fmt.Printf("Odebrane bajty [%d]: %v\n", n, buffer[:n])
}

func sendTCPData(address string, data []byte) {
	// Połącz się z serwerem
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Błąd podczas łączenia z serwerem: %v\n", err)
		return
	}
	defer conn.Close()

	// Wyślij dane
	_, err = conn.Write(data)
	if err != nil {
		fmt.Printf("Błąd podczas wysyłania danych: %v\n", err)
		return
	}

	// Wyświetl wysłane bajty
	fmt.Printf("Wysłane bajty: %v\n", data)
}
