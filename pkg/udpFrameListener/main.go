package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"frame_reductor/model"
	"net"
	"os"
	"time"
)

func main() {
	// Definicja flag
	port := flag.Int("port", 12345, "Port number to listen on")
	timeout := flag.Int("timeout", 60, "Timeout in seconds for the server to stop listening")
	flag.Parse()

	// Nasłuchiwanie na porcie UDP
	address := fmt.Sprintf(":%d", *port)
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		fmt.Printf("Błąd podczas tworzenia nasłuchiwania na porcie %d: %v\n", *port, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("Serwer nasłuchuje na porcie %d przez %d sekund...\n", *port, *timeout)

	// Kanał timeout
	serverTimeout := time.After(time.Duration(*timeout) * time.Second)

	// Obsługa przychodzących ramek UDP
	for {
		select {
		case <-serverTimeout:
			fmt.Println("Czas nasłuchiwania minął. Zamykam serwer...")
			return
		default:
			// Odczyt ramek UDP
			err := handleUDPConnection(conn)
			if err != nil {
				fmt.Printf("Błąd podczas obsługi połączenia UDP: %v\n", err)
			}
		}
	}
}

func handleUDPConnection(conn net.PacketConn) error {
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	// Odczyt danych UDP
	frameData, err := readUDPFrame(conn)
	if err != nil {
		// Sprawdzenie, czy błąd wynika z timeoutu
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("Timeout odczytu danych UDP. Oczekiwanie na kolejne ramki...")
			return nil
		}
		return fmt.Errorf("błąd podczas odczytu danych UDP: %w", err)
	}

	fmt.Printf("Odebrano ramkę [%d bytes]: %x\n", len(frameData), frameData)
	fmt.Printf("Odebrano ramkę [%d bytes]: %v\n", len(frameData), frameData)

	// Dekodowanie nagłówka
	header, err := model.DecodeC37Header(frameData[:14])
	if err != nil {
		fmt.Println("Błąd dekodowania nagłówka:", err)
		return nil
	}
	fmt.Printf("Header: %v\n", header)

	// Obsługa różnych typów ramek
	switch header.DataFrameType {
	case model.ConfigurationFrame2:
		// Dekodowanie ramki konfiguracyjnej 2
		model.CfgFrame2, err = model.DecodeConfigurationFrame2(frameData[14:], *header)
		if err != nil {
			fmt.Println("Błąd dekodowania ramki konfiguracyjnej 2:", err)
			return nil
		}
		fmt.Printf("Zdekodowana ramka konfiguracyjna 2: %+v\n", model.CfgFrame2)

	case model.ConfigurationFrame3:
		// Dekodowanie ramki konfiguracyjnej 3
		model.CfgFrame3, err = model.DecodeConfigurationFrame3(frameData[14:], *header)
		if err != nil {
			fmt.Println("Błąd dekodowania ramki konfiguracyjnej 3:", err)
			return nil
		}
		fmt.Printf("Zdekodowana ramka konfiguracyjna 3: %+v\n", model.CfgFrame3)

	case model.DataFrame:
		// Sprawdzenie, czy ramka konfiguracyjna jest dostępna
		if model.CfgFrame2 == nil && model.CfgFrame3 == nil {
			fmt.Println("Brak ramki konfiguracyjnej. Pomijam ramkę danych.")
			return nil
		}

		// Dekodowanie ramki z danymi
		dataFrame, err := model.DecodeDataFrame(frameData[14:], *header)
		if err != nil {
			fmt.Println("Błąd dekodowania ramki z danymi:", err)
			return nil
		}
		fmt.Printf("Zdekodowana ramka danych: %+v\n", dataFrame)
	default:
		fmt.Printf("Nieznany typ ramki: %v\n", header.DataFrameType)
	}

	return nil
}

func readUDPFrame(conn net.PacketConn) ([]byte, error) {
	// Nagłówek ramki ma co najmniej 4 bajty
	header := make([]byte, 4)
	n, addr, err := conn.ReadFrom(header)
	if err != nil {
		// Obsługa timeoutu
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("Timeout odczytu danych UDP. Oczekiwanie na kolejne ramki...")
			return nil, nil
		}
		return nil, fmt.Errorf("błąd podczas odczytu nagłówka ramki: %w", err)
	}

	if n < 4 {
		return nil, fmt.Errorf("odebrano zbyt mało bajtów na nagłówek: %d", n)
	}

	// Długość ramki określona w bajtach 3 i 4
	frameLength := int(binary.BigEndian.Uint16(header[2:4]))
	if frameLength < 4 {
		return nil, fmt.Errorf("nieprawidłowa długość ramki: %d", frameLength)
	}

	// Odczyt pozostałych danych ramki
	remainingBytes := frameLength - 4
	frameData := make([]byte, remainingBytes)
	n, _, err = conn.ReadFrom(frameData)
	if err != nil {
		return nil, fmt.Errorf("błąd podczas odczytu danych ramki: %w", err)
	}

	if n != remainingBytes {
		return nil, fmt.Errorf("niezgodna długość ramki: oczekiwano %d bajtów, odebrano %d", remainingBytes, n)
	}

	// Połącz nagłówek i dane w jedną ramkę
	fullFrame := append(header, frameData...)
	fmt.Printf("Odebrano ramkę [%d bytes] od %v: %x\n", len(fullFrame), addr, fullFrame)
	return fullFrame, nil
}
