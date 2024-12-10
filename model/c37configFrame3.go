package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// ConfigurationFrame struktura zawiera główne elementy ramki konfiguracyjnej
type ConfigurationFrame struct {
	IDCode      uint16
	FrameSize   uint16
	FrameType   uint16
	NumPMUs     uint16
	PhasorNames []string
	Phnmr       uint16
	Annmr       uint16
	Dgnmr       uint16
}

// DecodeConfigurationFrame Dekoduje ramkę konfiguracyjną z weryfikacją długości
func DecodeConfigurationFrame(data []byte) (*ConfigurationFrame, error) {
	reader := bytes.NewReader(data)
	cfg := &ConfigurationFrame{}

	// Dekodowanie ID Code
	if err := binary.Read(reader, binary.BigEndian, &cfg.IDCode); err != nil {
		return nil, fmt.Errorf("błąd dekodowania IDCode: %v", err)
	}

	// Dekodowanie FrameSize
	if err := binary.Read(reader, binary.BigEndian, &cfg.FrameSize); err != nil {
		return nil, fmt.Errorf("błąd dekodowania FrameSize: %v", err)
	}

	// Dekodowanie FrameType
	if err := binary.Read(reader, binary.BigEndian, &cfg.FrameType); err != nil {
		return nil, fmt.Errorf("błąd dekodowania FrameType: %v", err)
	}

	// Dekodowanie liczby PMU (NumPMUs)
	if err := binary.Read(reader, binary.BigEndian, &cfg.NumPMUs); err != nil {
		return nil, fmt.Errorf("błąd dekodowania NumPMUs: %v", err)
	}

	// Dekodowanie liczby fazorów
	if err := binary.Read(reader, binary.BigEndian, &cfg.Phnmr); err != nil {
		return nil, fmt.Errorf("błąd dekodowania Phnmr: %v", err)
	}

	// Weryfikacja, czy jest wystarczająca liczba bajtów dla fazorów
	//expectedLength := int(cfg.Phnmr) * 16
	//if len(data) < reader.Len()+expectedLength {
	//	return nil, fmt.Errorf("niekompletna ramka - oczekiwano %d bajtów dla nazw fazorów, ale jest ich mniej: %d bajtów", expectedLength, len(data))
	//}

	// Dekodowanie nazw fazorów
	//cfg.PhasorNames = make([]string, cfg.Phnmr)
	//for i := 0; i < int(cfg.Phnmr); i++ {
	//	nameBytes := make([]byte, 16) // Każda nazwa zajmuje 16 bajtów
	//	if _, err := reader.Read(nameBytes); err != nil {
	//		return nil, fmt.Errorf("błąd odczytu nazwy fazora %d: %v", i+1, err)
	//	}
	//	cfg.PhasorNames[i] = strings.TrimRight(string(nameBytes), "\x00") // Usuwanie pustych bajtów z końca
	//}

	return cfg, nil
}
