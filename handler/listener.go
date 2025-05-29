package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"frame_reductor/model"
	"net"
	"os"
	"sync"
	"time"
)

type PMUFrame struct {
	IDCode   uint16
	SOC      uint32
	FracSec  uint32
	FrameRaw []byte
}

var (
	frameBuffer = make(map[string][]PMUFrame) // klucz = timestamp (SOC+FracSec)
	bufferMutex sync.Mutex
)

// StartListening - funkcja dla trybu "listen"
func StartListening(port, period int, outputFilename string, frameChan chan []byte) {
	// Określenie trybu zapisu ramek do pliku
	saveToFile := len(outputFilename) > 0

	// Nasłuch na wskazanym porcie UDP, domyślnie 4716
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
			fmt.Println("Czas nasłuchu upłynął.")
			break loop
		default:
			// Odczyt ramki UDP
			conn.SetReadDeadline(time.Now().Add(1 * time.Second)) // Timeout na odczyt
			frameData, err := ReadUDPFrame(conn)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // kontynuuj nasłuch po timeout
				}
				fmt.Println("Błąd podczas odczytu ramki:", err)
				break loop
			}

			// Konwersja ramki do formatu hex
			hexFrame := hex.EncodeToString(frameData)

			if saveToFile {
				// Zapisujemy ramkę do pliku
				_, err = file.WriteString(hexFrame + "\n")
				if err != nil {
					fmt.Println("Błąd podczas zapisu do pliku:", err)
					break loop
				}
			}
			fmt.Println("Odebrana ramka hex:", hexFrame)

			// Opcjonalne: Dekodowanie nagłówka
			if len(frameData) >= 14 {
				header, err := model.DecodeC37Header(frameData[:14])
				if err != nil {
					fmt.Println("Błąd dekodowania nagłówka:", err)
				} else {
					fmt.Printf("Header: %v\n", header)
				}

				soc := header.Soc
				frac := header.FracSec
				idCode := header.IDCode

				key := fmt.Sprintf("%d:%d", soc, frac)
				newFrame := PMUFrame{
					IDCode:   idCode,
					SOC:      soc,
					FracSec:  frac,
					FrameRaw: append([]byte(nil), frameData...), // kopia
				}

				bufferMutex.Lock()
				frameBuffer[key] = append(frameBuffer[key], newFrame)
				bufferMutex.Unlock()

				switch header.DataFrameType {
				case model.ConfigurationFrame2:
					// Dekodowanie ramki konfiguracyjnej 2
					model.CfgFrame2, err = model.DecodeConfigurationFrame2(frameData[14:], *header)
					if err != nil {
						fmt.Println("Błąd dekodowania ramki konfiguracyjnej 2:", err)
						return
					}
					fmt.Printf("Zdekodowana ramka konfiguracyjna 2: %+v\n", model.CfgFrame2)
					// Obsługa agregacji
					HandleConfigFrame(model.CfgFrame2, frameData, frameChan)
				case model.ConfigurationFrame3:
					// Dekodowanie ramki konfiguracyjnej 3
					model.CfgFrame3, err = model.DecodeConfigurationFrame3(frameData[14:], *header)
					if err != nil {
						fmt.Println("Błąd dekodowania ramki konfiguracyjnej 3:", err)
						return
					}
					fmt.Printf("Zdekodowana ramka konfiguracyjna 3: %+v\n", model.CfgFrame3)
				case model.DataFrame:
					// Do poprawnego zdekodowania ramki z danymi potrzebna jest ramka konfiguracyjna
					if model.CfgFrame2 == nil && model.CfgFrame3 == nil {
						continue
					}
					// Dekodowanie ramki z danymi
					dataFrame, err := model.DecodeDataFrame(frameData[14:], *header)
					if err != nil {
						fmt.Println("Błąd dekodowania ramki z danymi:", err)
						return
					}
					//fmt.Printf("Zdekodowana ramka danych: %+v\n", dataFrame)
					ProcessDataFrame(*dataFrame, frameData, frameChan)
				}
			}
		}
	}

	if saveToFile {
		fmt.Printf("Nasłuch zakończony, ramki zapisane do pliku %s.\n", outputFilename)
	} else {
		fmt.Println("Nasłuch zakończony.")
	}
}

// ReadUDPFrame odczytuje ramkę UDP z podanego połączenia
func ReadUDPFrame(conn net.PacketConn) ([]byte, error) {
	// Początkowy bufor o rozmiarze 1024 bajtów
	buffer := make([]byte, 1024)

	// Odczyt danych z połączenia
	n, _, err := conn.ReadFrom(buffer)
	if err != nil {
		// Obsługa timeoutu
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("Timeout odczytu danych UDP. Oczekiwanie na kolejne ramki...")
			return nil, nil
		}
		return nil, fmt.Errorf("błąd podczas odczytu danych UDP: %w", err)
	}

	fmt.Printf("\nOdebrano dane [%d bytes]: %X\n", n, buffer[:n])

	if n < 4 {
		return nil, fmt.Errorf("odebrano zbyt mało bajtów na nagłówek: %d", n)
	}

	// Długość ramki określona w bajtach 3 i 4
	frameLength := int(binary.BigEndian.Uint16(buffer[2:4]))
	if frameLength < 4 {
		return nil, fmt.Errorf("nieprawidłowa długość ramki: %d", frameLength)
	}

	if frameLength > n {
		return nil, fmt.Errorf("długość ramki (%d) przekracza odebraną ilość danych (%d)", frameLength, n)
	}

	// Tworzymy nowy slice o dokładnej długości i pojemności
	fullFrame := make([]byte, frameLength)
	copy(fullFrame, buffer[:frameLength])

	//fmt.Printf("Odebrano ramkę [%d bytes] od %v: %X\n", len(fullFrame), addr, fullFrame)
	return fullFrame, nil
}
