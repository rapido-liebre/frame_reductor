package handler

import (
	"fmt"
	"frame_reductor/model"
	"net"
)

// ProcessConfigurationFrame redukuje liczbę fazorów i wysyła zmodyfikowaną ramkę konfiguracyjną na wybrany port
func ProcessConfigurationFrame(frame model.C37ConfigurationFrame2, frameData []byte) {
	// Wypisz dane ramki
	fmt.Printf("Dane ramki: %+v\n", frame)
	fmt.Printf("Ramka konfiguracyjna: %+v\n", frameData)

	// Wyślij ramkę konfiguracyjną na odpowiedni port
	if model.Out.Protocol != "" && model.Out.Port != 0 {
		frameConverted, frameDataConverted, err := ConvertConfigurationFrame(frame, frameData)
		if err != nil {
			fmt.Printf("Błąd konwersji ramki konfiguracyjnej: %v\n", err)
		}
		fmt.Printf("Ramka do wysłania [%d bytes]: %v\n", len(frameDataConverted), frameConverted)

		err = sendFrame(model.Out.Protocol, model.Out.Port, frameDataConverted)
		//time.Sleep(10 * time.Minute)
		if err != nil {
			fmt.Printf("Błąd wysyłania ramki konfiguracyjnej: %v\n", err)
		} else {
			fmt.Println("Ramka konfiguracyjna została wysłana.")
		}
	} else {
		fmt.Println("Protokół lub port nie są zdefiniowane. Ramka konfiguracyjna nie została wysłana.")
	}
}

// ProcessDataFrame redukuje liczbę fazorów i wysyła zmodyfikowaną ramkę danych na wybrany port
func ProcessDataFrame(frame model.C37DataFrame, frameData []byte) {
	// Oblicz interwał
	interval := model.CfgFrame2.TimeBase.TimeMultiplier / model.FramesCount

	// Sprawdź, czy FracSec jest wielokrotnością interwału
	if frame.FracSec%interval == 0 {
		// Wypisz dane ramki
		fmt.Printf("Dane ramki: %+v\n", frame)
		fmt.Printf("Ramka danych: %+v\n", frameData)

		// Wyślij ramkę danych na odpowiedni port
		if model.Out.Protocol != "" && model.Out.Port != 0 {
			frameConverted, frameDataConverted, err := ConvertDataFrame(frame, frameData)
			if err != nil {
				fmt.Printf("Błąd konwersji ramki danych: %v\n", err)
			}
			fmt.Printf("Ramka do wysłania [%d bytes]: %v\n", len(frameDataConverted), frameConverted)

			err = sendFrame(model.Out.Protocol, model.Out.Port, frameData)
			if err != nil {
				fmt.Printf("Błąd wysyłania ramki danych: %v\n", err)
			} else {
				fmt.Println("Ramka danych została wysłana.")
			}
		} else {
			fmt.Println("Protokół lub port nie są zdefiniowane. Ramka danych nie została wysłana.")
		}
	} else {
		fmt.Println("Ramka danych nie spełnia warunku wielokrotności. ", frame.FracSec)
	}
}

func sendFrame(protocol model.Protocol, port uint32, frameData []byte) error {
	address := fmt.Sprintf("localhost:%d", port) // Zakładamy wysyłanie na localhost

	switch protocol {
	case model.ProtocolUDP:
		conn, err := net.Dial("udp", address)
		if err != nil {
			return fmt.Errorf("błąd połączenia UDP: %v", err)
		}
		defer conn.Close()

		_, err = conn.Write(frameData)
		if err != nil {
			return fmt.Errorf("błąd wysyłania danych przez UDP: %v", err)
		}

	case model.ProtocolTCP:
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("błąd połączenia TCP: %v", err)
		}
		defer conn.Close()

		n, err := conn.Write(frameData)
		if err != nil {
			return fmt.Errorf("błąd wysyłania danych przez TCP: %v", err)
		}
		if n != len(frameData) {
			return fmt.Errorf("Nie wysłano wszystkich danych: wysłano %d z %d bajtów", n, len(frameData))
		}

	default:
		return fmt.Errorf("nieznany protokół: %v", protocol)
	}

	return nil
}
