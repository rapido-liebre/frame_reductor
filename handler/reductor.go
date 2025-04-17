package handler

import (
	"fmt"
	"frame_reductor/model"
	"net"
	"time"
)

// ProcessConfigurationFrame redukuje liczbę fazorów i wysyła zmodyfikowaną ramkę konfiguracyjną na wybrany port
func ProcessConfigurationFrame(frame model.C37ConfigurationFrame2, frameData []byte, frameChan chan []byte) {
	// Wypisz dane ramki
	fmt.Printf("Dane ramki: %+v\n", frame)
	fmt.Printf("Ramka konfiguracyjna: %+v\n", frameData)

	// Wyślij ramkę konfiguracyjną na odpowiedni port
	if model.Out.Protocol != "" && model.Out.Port != 0 {
		frameConverted, frameDataConverted, err := ConvertConfigurationFrame(frame, frameData)
		if err != nil {
			fmt.Printf("Błąd konwersji ramki konfiguracyjnej: %v\n", err)
		}
		fmt.Printf("Ramka do wysłania [%d bytes]: %v\n[%+v]", len(frameDataConverted), frameConverted, frameDataConverted)

		err = sendFrame(model.Out.Protocol, model.Out.Port, frameDataConverted, frameChan)
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
func ProcessDataFrame(frame model.C37DataFrame, frameData []byte, frameChan chan []byte) {
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
			fmt.Printf("Ramka do wysłania [%d bytes]: %v\n[%+v]\n", len(frameDataConverted), frameConverted, frameDataConverted)

			err = sendFrame(model.Out.Protocol, model.Out.Port, frameDataConverted, frameChan)
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

func sendFrame(protocol model.Protocol, port uint32, frameData []byte, frameChan chan []byte) error {
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
		switch model.Out.TCPMode {
		case model.TCPServer:
			fmt.Printf("Wysyłam ramkę do kanału. %v\n", frameData)
			frameChan <- frameData
		case model.TCPClient:
			// Niezależnie od trybu (server/client), wysyłamy do kanału
			select {
			case frameChan <- frameData:
				fmt.Printf("Wysłano ramkę do kanału TCP [%d bytes]\n", len(frameData))
			case <-time.After(1 * time.Second):
				return fmt.Errorf("timeout: nie udało się wysłać ramki do kanału")
			}
		}

	default:
		return fmt.Errorf("nieznany protokół: %v", protocol)
	}

	return nil
}
