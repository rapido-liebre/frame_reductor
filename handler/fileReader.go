package handler

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"frame_reductor/model"
	"os"
	"path/filepath"
)

// ProcessFile - funkcja dla trybu "file"
func ProcessFile(frameChan chan []byte) {
	// Ustaw katalog roboczy jako katalog główny projektu
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Błąd pobierania katalogu roboczego:", err)
		return
	}
	// Przejdź o dwa poziomy w górę
	projectRoot := filepath.Join(workingDir, "..", "..")
	fmt.Println("Katalog główny projektu:", projectRoot)

	filePath := filepath.Join(projectRoot, "udp_frames_ROG_02.01.txt")

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Błąd otwierania pliku:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Konwersja linii z formatu hex na []byte
		frameData, err := hex.DecodeString(line)
		if err != nil {
			fmt.Println("Błąd dekodowania hex:", err)
			continue
		}

		header, err := model.DecodeC37Header(frameData[:14])
		if err != nil {
			fmt.Println("Błąd dekodowania nagłówka:", err)
			return
		}
		//fmt.Printf("Header: %v\n", header)

		switch header.DataFrameType {
		case model.ConfigurationFrame2:
			// Dekodowanie ramki konfiguracyjnej 2
			model.CfgFrame2, err = model.DecodeConfigurationFrame2(frameData[14:], *header)
			if err != nil {
				fmt.Println("Błąd dekodowania ramki konfiguracyjnej 2:", err)
				return
			}
			fmt.Printf("Zdekodowana ramka konfiguracyjna 2: %+v\n", model.CfgFrame2)
			ProcessConfigurationFrame(*model.CfgFrame2, frameData, frameChan)
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
	// Wyświetlenie informacji o ramce konfiguracyjnej
	//fmt.Printf("Configuration Frame:\n")
	//fmt.Printf("ID Code: %d\n", cfg.IDCode)
	//fmt.Printf("Frame Size: %d\n", cfg.FrameSize)
	//fmt.Printf("Frame Type: %d\n", cfg.FrameType)
	//fmt.Printf("Num PMUs: %d\n", cfg.NumPMUs)
	//fmt.Printf("DataPhasor Names:\n")
	//for i, name := range cfg.PhasorNames {
	//	fmt.Printf("  DataPhasor %d: %s\n", i+1, name)
	//}
}
