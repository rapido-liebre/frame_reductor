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
	C37Header
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

func DecodeConfigurationFrame2(data []byte, header C37Header) (*C37ConfigurationFrame2, error) {
	reader := bytes.NewReader(data)
	var frame2 C37ConfigurationFrame2

	frame2.C37Header = header

	// Odczyt TimeBase
	var timeBase uint32
	if err := binary.Read(reader, binary.BigEndian, &timeBase); err != nil {
		return nil, fmt.Errorf("Błąd odczytu TimeBase: %v", err)
	}
	// Dekodowanie bitów pola TimeBase
	frame2.TimeBase = DecodeTimeBase(timeBase)

	if err := binary.Read(reader, binary.BigEndian, &frame2.NumPMU); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumPMU: %v", err)
	}

	// Odczyt nazwy stacji
	stationName := make([]byte, 16)
	if err := binary.Read(reader, binary.BigEndian, &stationName); err != nil {
		return nil, fmt.Errorf("Błąd odczytu StationName: %v", err)
	}
	// Konwertuj na string i usuń null bajty
	frame2.StationName = strings.TrimRight(string(stationName), "\x00")

	if err := binary.Read(reader, binary.BigEndian, &frame2.IDCode2); err != nil {
		return nil, fmt.Errorf("Błąd odczytu IDCode: %v", err)
	}

	// Format danych
	var format uint16
	if err := binary.Read(reader, binary.BigEndian, &format); err != nil {
		return nil, fmt.Errorf("Błąd odczytu Format: %v", err)
	}
	// Dekodowanie bitów pola FORMAT
	frame2.Format = DecodeFormatBits(format)

	// Dekodowanie liczby fazorów, analogów i cyfrowych słów statusu
	if err := binary.Read(reader, binary.BigEndian, &frame2.NumPhasors); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumPhasors: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame2.NumAnalogs); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumAnalogs: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame2.NumDigitals); err != nil {
		return nil, fmt.Errorf("Błąd odczytu NumDigitals: %v", err)
	}

	// Dekodowanie nazw kanałów
	channelNames, err := DecodeChannelNames(reader, frame2.NumPhasors, frame2.NumAnalogs, frame2.NumDigitals)
	if err != nil {
		log.Printf("Błąd odczytu ChannelNames: %v", err)
		return nil, err
	}
	log.Printf("Odczytane nazwy kanałów: %v", channelNames)
	frame2.ChannelNames = channelNames

	// Dekodowanie jednostek dla fazorów
	phasorUnits, err := DecodePhasorUnits(reader, frame2.NumPhasors)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu PhasorUnit: %v", err)
	}
	frame2.PhasorUnits = phasorUnits

	// Dekodowanie jednostek dla analogów
	analogUnits, err := DecodeAnalogUnits(reader, frame2.NumAnalogs)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu AnalogUnit: %v", err)
	}
	frame2.AnalogUnits = analogUnits

	// Dekodowanie masek cyfrowych
	digitalUnits, err := DecodeDigitalUnits(reader, frame2.NumDigitals)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu DigitalUnit: %v", err)
	}
	frame2.DigitalUnits = digitalUnits

	fNom, err := DecodeFreqNominal(reader)
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu FrequencyNominal: %v", err)
	}
	frame2.FNom = *fNom
	//fmt.Printf("Read FreqNom: %d (bytes: %x)\n", frame2.FNom, frame2.FNom)

	//remainingBytes := reader.Len()
	//fmt.Printf("Pozostałe bajty w reader: %d\n", remainingBytes)

	if err := binary.Read(reader, binary.BigEndian, &frame2.ConfigCount); err != nil {
		return nil, fmt.Errorf("Błąd odczytu ConfigCount: %v", err)
	}
	fmt.Printf("Read ConfigCount: %d (bytes: %x)\n", frame2.ConfigCount, frame2.ConfigCount)

	//remainingBytes = reader.Len()
	//fmt.Printf("Pozostałe bajty w reader: %d\n", remainingBytes)

	if err := binary.Read(reader, binary.BigEndian, &frame2.DataRate); err != nil {
		return nil, fmt.Errorf("Błąd odczytu DataRate: %v", err)
	}
	InputDataRate = float64(frame2.DataRate)
	//fmt.Printf("Read DataRate: %d (bytes: %x)\n", frame2.DataRate, frame2.DataRate)

	if err := binary.Read(reader, binary.BigEndian, &frame2.CRC); err != nil {
		return nil, fmt.Errorf("Błąd odczytu sumy kontrolnej CRC: %v", err)
	}

	return &frame2, nil
}
