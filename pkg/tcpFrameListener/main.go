package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"frame_reductor/model"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	// Definicja flag
	port := flag.Int("port", 12345, "Port number to listen on")
	timeout := flag.Int("timeout", 60, "Timeout in seconds for the server to stop listening")
	flag.Parse()

	// Nasłuchiwanie na porcie TCP
	address := fmt.Sprintf(":%d", *port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Błąd podczas tworzenia nasłuchiwania na porcie %d: %v\n", *port, err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("Serwer nasłuchuje na porcie TCP %d przez %d sekund...\n", *port, *timeout)

	// Kanał timeout
	serverTimeout := time.After(time.Duration(*timeout) * time.Second)

	// Obsługa przychodzących połączeń
	for {
		select {
		case <-serverTimeout:
			fmt.Println("Czas nasłuchiwania minął. Zamykam serwer...")
			return
		default:
			// Akceptowanie nowego połączenia
			conn, err := listener.Accept()
			if err != nil {
				fmt.Printf("Błąd podczas akceptacji połączenia: %v\n", err)
				continue
			}

			// Obsługa połączenia
			go handleTcpConnection(conn)
		}
	}
}

func handleTcpConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Połączono z klientem: %v\n", conn.RemoteAddr())

	//scanner := bufio.NewScanner(conn)
	idleTimeout := time.NewTimer(5 * time.Second)

	for {
		select {
		case <-idleTimeout.C:
			fmt.Println("Brak nowych ramek. Zamykam połączenie.")
			return
		default:
			// Ustawienie timeoutu dla odczytu
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			// Odczyt ramki za pomocą readFrame
			frameData, err := readFrame(conn)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Połączenie zakończone przez klienta.")
					return
				}
				fmt.Printf("Błąd odczytu ramki: %v\n", err)
				return
			}

			// Reset timeoutu po odebraniu ramki
			idleTimeout.Reset(5 * time.Second)

			// Wyświetlenie odebranych danych
			fmt.Printf("Odebrano ramkę [%d bytes]: %x\n[%+v]", len(frameData), frameData, frameData)

			// Dekodowanie nagłówka
			header, err := model.DecodeC37Header(frameData[:14])
			if err != nil {
				fmt.Println("Błąd dekodowania nagłówka:", err)
				return
			}
			fmt.Printf("Header: %v\n", header)

			// Obsługa różnych typów ramek
			switch header.DataFrameType {
			case model.ConfigurationFrame2:
				// Dekodowanie ramki konfiguracyjnej 2
				model.CfgFrame2, err = model.DecodeConfigurationFrame2(frameData[14:], *header)
				if err != nil {
					fmt.Println("Błąd dekodowania ramki konfiguracyjnej 2:", err)
					return
				}
				fmt.Printf("Zdekodowana ramka konfiguracyjna 2: %+v\n", model.CfgFrame2)

			case model.ConfigurationFrame3:
				// Dekodowanie ramki konfiguracyjnej 3
				model.CfgFrame3, err = model.DecodeConfigurationFrame3(frameData[14:], *header)
				if err != nil {
					fmt.Println("Błąd dekodowania ramki konfiguracyjnej 3:", err)
					return
				}
				fmt.Printf("Zdekodowana ramka konfiguracyjna 3: %+v\n", model.CfgFrame3)

			case model.DataFrame:
				// Sprawdzenie, czy ramka konfiguracyjna jest dostępna
				if model.CfgFrame2 == nil && model.CfgFrame3 == nil {
					fmt.Println("Brak ramki konfiguracyjnej. Pomijam ramkę danych.")
					continue
				}

				// Dekodowanie ramki z danymi
				dataFrame, err := model.DecodeDataFrame(frameData[14:], *header)
				if err != nil {
					fmt.Println("Błąd dekodowania ramki z danymi:", err)
					return
				}
				fmt.Printf("Zdekodowana ramka danych: %+v\n", dataFrame)
			default:
				fmt.Printf("Nieznany typ ramki: %v\n", header.DataFrameType)
			}
		}
	}
}

func readFrame(conn net.Conn) ([]byte, error) {
	// Nagłówek ramki ma 4 bajty
	header := make([]byte, 4)
	n, err := conn.Read(header)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu nagłówka ramki (odczytano %d bajtów): %v", n, err)
	}
	if n < 4 {
		return nil, fmt.Errorf("odebrano zbyt mało bajtów na nagłówek: %d", n)
	}

	// Długość ramki określona w bajtach 3 i 4
	frameLength := int(binary.BigEndian.Uint16(header[2:4]))
	if frameLength < 4 {
		return nil, fmt.Errorf("nieprawidłowa długość ramki: %d (musi być >= 4)", frameLength)
	}

	// Przygotowanie bufora na pozostałe dane
	remainingBytes := frameLength - 4
	buffer := make([]byte, remainingBytes)
	totalRead := 0

	// Odczyt pozostałych danych ramki
	for totalRead < remainingBytes {
		n, err := conn.Read(buffer[totalRead:])
		if err != nil {
			return nil, fmt.Errorf("błąd odczytu danych ramki (odczytano %d bajtów): %v", totalRead, err)
		}
		if n == 0 {
			break // Koniec strumienia
		}
		totalRead += n
	}

	if totalRead != remainingBytes {
		return nil, fmt.Errorf("niezgodna długość ramki: oczekiwano %d bajtów, odebrano %d", remainingBytes, totalRead)
	}

	// Debugowanie: opcjonalnie loguj pełną ramkę
	fullFrame := append(header, buffer...)
	fmt.Printf("Odebrano ramkę [%d bytes]: %x\n", len(fullFrame), fullFrame)

	// Zwracamy pełną ramkę (nagłówek + dane)
	return fullFrame, nil
}

func readFrame2(conn net.Conn) ([]byte, error) {
	// Nagłówek ramki ma 4 bajty
	header := make([]byte, 4)
	n, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu nagłówka ramki (odczytano %d bajtów): %v", n, err)
	}

	// Długość ramki określona w bajtach 3 i 4
	frameLength := int(binary.BigEndian.Uint16(header[2:4]))
	if frameLength < 4 {
		return nil, fmt.Errorf("nieprawidłowa długość ramki: %d (musi być >= 4)", frameLength)
	}

	// Odczyt pozostałych danych ramki
	frameData := make([]byte, frameLength-4) // Odejmujemy już odczytane 4 bajty nagłówka
	n, err = io.ReadFull(conn, frameData)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu danych ramki (odczytano %d bajtów): %v", n, err)
	}

	// Debugowanie: opcjonalnie loguj pełną ramkę
	fullFrame := append(header, frameData...)
	fmt.Printf("Odebrano ramkę [%d bytes]: %x\n", len(fullFrame), fullFrame)

	// Zwracamy pełną ramkę (nagłówek + dane)
	return fullFrame, nil
}

//func readFrame(conn net.Conn) ([]byte, error) {
//	// Nagłówek ramki to co najmniej 4 bajty
//	header := make([]byte, 4)
//	_, err := io.ReadFull(conn, header)
//	if err != nil {
//		return nil, fmt.Errorf("błąd odczytu nagłówka ramki: %v", err)
//	}
//
//	// Długość ramki określona w bajtach 3 i 4
//	frameLength := int(binary.BigEndian.Uint16(header[2:4]))
//	if frameLength <= 0 {
//		return nil, fmt.Errorf("nieprawidłowa długość ramki: %d", frameLength)
//	}
//
//	// Odczyt pozostałych danych ramki
//	frameData := make([]byte, frameLength-4) // Odejmujemy już odczytane 4 bajty nagłówka
//	_, err = io.ReadFull(conn, frameData)
//	if err != nil {
//		return nil, fmt.Errorf("błąd odczytu danych ramki: %v", err)
//	}
//
//	// Zwracamy pełną ramkę (nagłówek + dane)
//	return append(header, frameData...), nil
//}
