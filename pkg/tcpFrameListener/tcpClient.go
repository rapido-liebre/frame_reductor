package main

import (
	"fmt"
	"io"
	"net"
	"time"
)

func StartTCPClient(port int) {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Błąd podczas łączenia z serwerem TCP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Połączono z serwerem TCP na porcie %d\n", port)

	idleTimeout := time.NewTimer(10 * time.Second)

	for {
		select {
		case <-idleTimeout.C:
			fmt.Println("Brak danych od serwera. Zamykam połączenie.")
			return
		default:
			// Odczyt ramki
			frameData := make([]byte, 1024)
			n, err := conn.Read(frameData)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Połączenie zamknięte przez serwer.")
					return
				}
				fmt.Println("Błąd odczytu danych:", err)
				return
			}

			// Reset timeoutu po odebraniu danych
			idleTimeout.Reset(10 * time.Second)

			fmt.Printf("Odebrano ramkę [%d bytes]: %x\n", n, frameData[:n])
		}
	}
}
