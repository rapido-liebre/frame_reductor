package handler

import (
	"context"
	"fmt"
	"net"
	"time"
)

func StartTCPServer(port int, frameChan chan []byte) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Tworzymy serwer TCP
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Błąd podczas tworzenia serwera TCP na porcie %d: %v\n", port, err)
		return
	}
	defer listener.Close()

	fmt.Printf("Serwer TCP nasłuchuje na porcie %d...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Błąd podczas akceptacji połączenia:", err)
			continue
		}

		// Obsługa połączenia w osobnej goroutine
		go handleTCPConnection(ctx, conn, frameChan)
	}
}

func handleTCPConnection(ctx context.Context, conn net.Conn, frameChan chan []byte) {
	defer conn.Close()
	fmt.Printf("Połączono z klientem: %v\n", conn.RemoteAddr())

	idleTimeout := time.NewTimer(10 * time.Second)

	// Wysyłanie ramek do klienta
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context zakończony, zamykam połączenie.")
			return

		case frame := <-frameChan:
			_, err := conn.Write(frame)
			if err != nil {
				fmt.Printf("Błąd wysyłania ramki do %v: %v\n", conn.RemoteAddr(), err)
				//return
			}
			fmt.Printf("Wysłano ramkę [%d bytes] do %v\n", len(frame), conn.RemoteAddr())

			// Resetujemy timeout po każdej wysłanej ramce
			idleTimeout.Reset(10 * time.Second)

		case <-idleTimeout.C:
			fmt.Println("Timeout bez danych. Zamykam połączenie.")
			return
		}
	}
}
