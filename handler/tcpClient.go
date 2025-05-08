package handler

import (
	"fmt"
	"net"
	"time"
)

func StartTCPClient(port uint32, targetHost, bindIP string, frameChan chan []byte) {
	address := fmt.Sprintf("%s:%d", targetHost, port)

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	if bindIP != "" {
		dialer.LocalAddr = &net.TCPAddr{
			IP: net.ParseIP(bindIP),
		}
	}

	for {
		conn, err := dialer.Dial("tcp", address)
		if err != nil {
			fmt.Println("Nie udało się połączyć z serwerem TCP, próba ponownie za 3 sekundy...")
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("Połączono z serwerem TCP.")
		sendFramesOverConnection(conn, frameChan)

		fmt.Println("Połączenie zakończone. Próba ponownego połączenia za 3 sekundy...")
		time.Sleep(3 * time.Second)
	}
}

func sendFramesOverConnection(conn net.Conn, frameChan chan []byte) {
	defer conn.Close()

	idleTimeout := time.NewTimer(10 * time.Second)

	for {
		select {
		case frame := <-frameChan:
			_, err := conn.Write(frame)
			if err != nil {
				fmt.Println("Błąd podczas wysyłania ramki:", err)
				return
			}
			fmt.Printf("Wysłano ramkę [%d bytes]\n", len(frame))
			idleTimeout.Reset(10 * time.Second)
		case <-idleTimeout.C:
			fmt.Println("Timeout bez danych. Zamykam połączenie.")
			return
		}
	}
}
