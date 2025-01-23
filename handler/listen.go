package handler

import (
	"encoding/hex"
	"fmt"
	"frame_reductor/model"
	"net"
	"os"
	"time"
)

// StartListening - funkcja dla trybu "listen"
func StartListening(port, period int, outputFilename string) {
	// Określenie trybu zapisu ramek do pliku
	saveToFile := len(outputFilename) > 0

	// Nasłuch na wskazanym porcie UDP, domyślnie 4716
	// Adres lokalny na wskazanym porcie
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}

	// Otwieramy gniazdo UDP
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Błąd podczas otwierania gniazda:", err)
		return
	}
	defer conn.Close()

	var file *os.File
	if saveToFile {
		// Otwieramy plik do zapisu ramek
		file, err = os.Create(outputFilename)
		if err != nil {
			fmt.Println("Błąd podczas tworzenia pliku:", err)
			return
		}
		defer file.Close()
	}

	// Ustawiamy czas zakończenia nasłuchu
	var timeout <-chan time.Time
	if period > 0 {
		timeout = time.After(time.Duration(period) * time.Second)
		fmt.Printf("Nasłuchuję ramki UDP przez %d sekund...\n", period)
	} else {
		fmt.Println("Nasłuchuję ramki UDP w trybie ciągłym...")
	}

loop:
	for {
		select {
		case <-timeout:
			if period > 0 {
				fmt.Println("Czas nasłuchu upłynął.")
				break loop
			}
		default:
			// Zwiększony rozmiar bufora, aby uniknąć błędów związanych z dużymi ramkami
			frame := make([]byte, 1024)
			// Odbieramy dane UDP
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, _, err := conn.ReadFromUDP(frame)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // kontynuuj nasłuch po timeout
				}
				fmt.Println("Błąd podczas odczytu ramki: ", err)
				fmt.Println("Wykryta długość ramki: ", n)
				break loop
			}

			// Konwersja ramki do formatu hex
			hexFrame := hex.EncodeToString(frame)

			if saveToFile {
				// Zapisujemy ramkę do pliku
				_, err = file.WriteString(hexFrame + "\n")
				if err != nil {
					fmt.Println("Błąd podczas zapisu do pliku:", err)
					break loop
				}
			}
			fmt.Println("Odebrana ramka hex:", hexFrame)

			header, err := model.DecodeC37Header(frame[:18])
			if err != nil {
				fmt.Println("Błąd dekodowania nagłówka:", err)
				return
			}
			fmt.Printf("Header: %v\n", header)
		}
	}

	if saveToFile {
		fmt.Printf("Nasłuch zakończony, ramki zapisane do pliku %s.", outputFilename)
	} else {
		fmt.Println("Nasłuch zakończony.")
	}
}
