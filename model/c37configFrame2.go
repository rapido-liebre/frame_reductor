package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
)

// C37ConfigurationFrame2 reprezentuje ramkę konfiguracji typu 2 dla standardu C37.118.
type C37ConfigurationFrame2 struct {
	Sync         uint16        `json:"sync"`          // Bajt synchronizujący z typem ramki i numerem wersji
	FrameSize    uint16        `json:"frame_size"`    // Liczba bajtów w ramce
	IDCode       uint16        `json:"id_code"`       // Główny identyfikator strumienia danych PMU/PDC
	SOC          uint32        `json:"soc"`           // Znacznik czasu SOC
	FracSec      uint32        `json:"frac_sec"`      // Ułamek sekundy i jakość znacznika czasu
	TimeBase     TimeBaseBits  `json:"time_base"`     // Rozdzielczość znacznika czasu FRACSEC
	NumPMU       uint16        `json:"num_pmu"`       // Liczba PMU zawartych w ramce danych
	StationName  string        `json:"station_name"`  // Nazwa stacji w formacie ASCII
	IDCode2      uint16        `json:"id_code_2"`     // Dodatkowy identyfikator strumienia danych PMU/PDC
	Format       FormatBits    `json:"format"`        // Format danych w ramce danych
	NumPhasors   uint16        `json:"num_phasors"`   // Liczba fazorów
	NumAnalogs   uint16        `json:"num_analogs"`   // Liczba wartości analogowych
	NumDigitals  uint16        `json:"num_digitals"`  // Liczba cyfrowych słów statusu
	ChannelNames []string      `json:"channel_names"` // Nazwy kanałów fazorów, analogowych i cyfrowych
	PhasorUnits  []PhasorUnit  `json:"phasor_units"`  // Współczynniki konwersji dla kanałów fazorów
	AnalogUnits  []AnalogUnit  `json:"analog_units"`  // Współczynniki konwersji dla kanałów analogowych
	DigitalUnits []DigitalUnit `json:"digital_units"` // Maski dla cyfrowych słów statusu
	FNom         FNom          `json:"f_nom"`         // Kod częstotliwości nominalnej i flagi
	ConfigCount  uint16        `json:"config_count"`  // Licznik zmian konfiguracji
	DataRate     int16         `json:"data_rate"`     // Szybkość transmisji danych fazorów, ilość ramek/sek
	CRC          uint16        `json:"crc"`           // Suma kontrolna CRC-CCITT
}

func DecodeConfigurationFrame2(data []byte) (*C37ConfigurationFrame2, error) {
	reader := bytes.NewReader(data)
	var header C37ConfigurationFrame2

	// Dekodowanie pól nagłówka
	if err := binary.Read(reader, binary.BigEndian, &header.Sync); err != nil {
		return nil, fmt.Errorf("Błąd odczytu SYNC: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.FrameSize); err != nil {
		return nil, fmt.Errorf("Błąd odczytu FrameSize: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.IDCode); err != nil {
		return nil, fmt.Errorf("Błąd odczytu IDCode: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.SOC); err != nil {
		return nil, fmt.Errorf("Błąd odczytu Soc: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.FracSec); err != nil {
		return nil, fmt.Errorf("Błąd odczytu FracSec: %v", err)
	}

	// Odczyt TimeBase
	var timeBase uint32
	if err := binary.Read(reader, binary.BigEndian, &timeBase); err != nil {
		return nil, fmt.Errorf("Błąd odczytu TimeBase: %v", err)
	}
	// Dekodowanie bitów pola TimeBase
	header.TimeBase = DecodeTimeBase(timeBase)

	if err := binary.Read(reader, binary.BigEndian, &header.NumPMU); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumPMU: %v", err)
	}

	// Odczyt nazwy stacji
	stationName := make([]byte, 16)
	if err := binary.Read(reader, binary.BigEndian, &stationName); err != nil {
		return nil, fmt.Errorf("Błąd odczytu StationName: %v", err)
	}
	// Konwertuj na string i usuń null bajty
	header.StationName = strings.TrimRight(string(stationName), "\x00")

	if err := binary.Read(reader, binary.BigEndian, &header.IDCode2); err != nil {
		return nil, fmt.Errorf("Błąd odczytu IDCode: %v", err)
	}

	// Format danych
	var format uint16
	if err := binary.Read(reader, binary.BigEndian, &format); err != nil {
		return nil, fmt.Errorf("Błąd odczytu Format: %v", err)
	}
	// Dekodowanie bitów pola FORMAT
	header.Format = decodeFormatBits(format)

	// Dekodowanie liczby fazorów, analogów i cyfrowych słów statusu
	if err := binary.Read(reader, binary.BigEndian, &header.NumPhasors); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumPhasors: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.NumAnalogs); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumAnalogs: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.NumDigitals); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumDigitals: %v", err)
	}

	// Dekodowanie nazw kanałów
	channelNames, err := DecodeChannelNames(reader, header.NumPhasors, header.NumAnalogs, header.NumDigitals)
	if err != nil {
		log.Printf("Błąd odczytu ChannelNames: %v", err)
		return nil, err
	}
	log.Printf("Odczytane nazwy kanałów: %v", channelNames)
	header.ChannelNames = channelNames

	// Dekodowanie jednostek dla fazorów
	phasorUnits, err := DecodePhasorUnits(reader, header.NumPhasors)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu PhasorUnit: %v", err)
	}
	header.PhasorUnits = phasorUnits

	// Dekodowanie jednostek dla analogów
	analogUnits, err := DecodeAnalogUnits(reader, header.NumAnalogs)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu AnalogUnit: %v", err)
	}
	header.AnalogUnits = analogUnits

	// Dekodowanie masek cyfrowych
	digitalUnits, err := DecodeDigitalUnits(reader, header.NumDigitals)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu DigitalUnit: %v", err)
	}
	header.DigitalUnits = digitalUnits

	fNom, err := DecodeFreqNominal(reader)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu FrequencyNominal: %v", err)
	}
	header.FNom = *fNom

	if err := binary.Read(reader, binary.BigEndian, &header.ConfigCount); err != nil {
		return nil, fmt.Errorf("Błąd odczytu ConfigCount: %v", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &header.DataRate); err != nil {
		return nil, fmt.Errorf("Błąd odczytu DataRate: %v", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &header.CRC); err != nil {
		return nil, fmt.Errorf("Błąd odczytu sumy kontrolnej CRC: %v", err)
	}

	return &header, nil
}
