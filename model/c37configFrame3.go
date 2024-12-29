package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// C37ConfigurationFrame3 reprezentuje ramkę konfiguracji typu 3 dla standardu C37.118.
type C37ConfigurationFrame3 struct {
	Sync         uint16              `json:"sync"`          // Bajt synchronizujący z typem ramki i numerem wersji
	FrameSize    uint16              `json:"frame_size"`    // Liczba bajtów w ramce
	IDCode       uint16              `json:"id_code"`       // Główny identyfikator strumienia danych PMU/PDC
	SOC          uint32              `json:"soc"`           // Znacznik czasu SOC
	FracSec      uint32              `json:"frac_sec"`      // Ułamek sekundy i jakość znacznika czasu
	ContIdx      uint16              `json:"cont_idx"`      // Indeks fragmentacji ramki
	TimeBase     TimeBaseBits        `json:"time_base"`     // Rozdzielczość znacznika czasu FRACSEC
	NumPMU       uint16              `json:"num_pmu"`       // Liczba PMU zawartych w ramce danych
	StationName  string              `json:"station_name"`  // Nazwa stacji w formacie ASCII
	IDCode2      uint16              `json:"id_code_2"`     // Dodatkowy identyfikator strumienia danych PMU/PDC
	GlobalPMUID  [16]byte            `json:"global_pmu_id"` // Globalny identyfikator PMU
	Format       FormatBits          `json:"format"`        // Format danych w ramce danych
	NumPhasors   uint16              `json:"num_phasors"`   // Liczba fazorów
	NumAnalogs   uint16              `json:"num_analogs"`   // Liczba wartości analogowych
	NumDigitals  uint16              `json:"num_digitals"`  // Liczba cyfrowych słów statusu
	ChannelNames []string            `json:"channel_names"` // Nazwy kanałów fazorów, analogowych i cyfrowych
	PhasorScales []PhasorScaleFactor `json:"phasor_scales"` // Współczynniki konwersji dla kanałów fazorów
	AnalogScales []AnalogScaleFactor `json:"analog_scales"` // Współczynniki konwersji dla kanałów analogowych
	DigitalMasks []DigitalMask       `json:"digital_masks"` // Maski dla cyfrowych słów statusu
	PMULatitude  float32             `json:"pmu_latitude"`  // Szerokość geograficzna PMU (WGS84)
	PMULongitude float32             `json:"pmu_longitude"` // Długość geograficzna PMU (WGS84)
	PMUElevation float32             `json:"pmu_elevation"` // Wysokość PMU nad poziomem morza (WGS84)
	ServiceClass byte                `json:"service_class"` // Klasa usługi (M lub P)
	Window       uint32              `json:"window"`        // Długość okna pomiarowego w mikrosekundach
	GroupDelay   uint32              `json:"group_delay"`   // Opóźnienie grupy faz w mikrosekundach
	FNom         uint16              `json:"f_nom"`         // Kod częstotliwości nominalnej i flagi
	DataRate     int16               `json:"data_rate"`     // Szybkość transmisji danych fazorów
	ConfigCount  uint16              `json:"config_count"`  // Licznik zmian konfiguracji
	CRC          uint16              `json:"crc"`           // Suma kontrolna CRC-CCITT
}

func DecodeConfigurationFrame3(data []byte) (*C37ConfigurationFrame3, error) {
	reader := bytes.NewReader(data)
	var header C37ConfigurationFrame3

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
	if err := binary.Read(reader, binary.BigEndian, &header.ContIdx); err != nil {
		return nil, fmt.Errorf("Błąd odczytu ContIdx: %v", err)
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
	stationNameLen, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("Błąd odczytu długości StationName: %v", err)
	}
	stationName := make([]byte, stationNameLen)
	if _, err := reader.Read(stationName); err != nil {
		return nil, fmt.Errorf("Błąd odczytu StationName: %v", err)
	}
	header.StationName = string(stationName)

	if err := binary.Read(reader, binary.BigEndian, &header.IDCode2); err != nil {
		return nil, fmt.Errorf("Błąd odczytu IDCode: %v", err)
	}

	// Odczyt globalnego ID PMU
	if err := binary.Read(reader, binary.BigEndian, &header.GlobalPMUID); err != nil {
		return nil, fmt.Errorf("Błąd odczytu GlobalPMUID: %v", err)
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

	// Oblicz całkowitą liczbę nazw
	totalNames := int(header.NumPhasors) + int(header.NumAnalogs) // + int(header.NumDigitals) //TODO

	// Dekodowanie nazw kanałów z przesunięciem o 2 bajty
	channelNames, err := DecodeCHNAMWithOffsetAndLength(reader, totalNames)
	if err != nil {
		log.Printf("Błąd odczytu ChannelNames: %v", err)
		return nil, err
	}
	log.Printf("Odczytane nazwy kanałów: %v", channelNames)
	header.ChannelNames = channelNames

	// Dekodowanie skal dla fazorów
	phasorScales, err := DecodePhasorScale(reader, int(header.NumPhasors))
	if err != nil {
		return nil, fmt.Errorf("Błąd dekodowania PhasorScale: %v", err)
	}
	header.PhasorScales = phasorScales

	// Dekodowanie skal dla analogów
	analogScales, err := DecodeAnalogScale(reader, int(header.NumAnalogs))
	if err != nil {
		return nil, fmt.Errorf("Błąd dekodowania AnalogScale: %v", err)
	}
	header.AnalogScales = analogScales

	// Dekodowanie masek cyfrowych
	digitalMasks, err := DecodeDigitalMask(reader, header.NumDigitals)
	if err != nil {
		return nil, fmt.Errorf("Błąd dekodowania DigitalMask: %v", err)
	}
	header.DigitalMasks = digitalMasks

	// Pozostałe pola konfiguracyjne
	if err := binary.Read(reader, binary.BigEndian, &header.PMULatitude); err != nil {
		return nil, fmt.Errorf("Błąd odczytu PMULatitude: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.PMULongitude); err != nil {
		return nil, fmt.Errorf("Błąd odczytu PMULongitude: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.PMUElevation); err != nil {
		return nil, fmt.Errorf("Błąd odczytu PMUElevation: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.ServiceClass); err != nil {
		return nil, fmt.Errorf("Błąd odczytu SVCClass: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.Window); err != nil {
		return nil, fmt.Errorf("Błąd odczytu Window: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.GroupDelay); err != nil {
		return nil, fmt.Errorf("Błąd odczytu GrpDly: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.FNom); err != nil {
		return nil, fmt.Errorf("Błąd odczytu FNom: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.DataRate); err != nil {
		return nil, fmt.Errorf("Błąd odczytu DataRate: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.ConfigCount); err != nil {
		return nil, fmt.Errorf("Błąd odczytu ConfigCount: %v", err)
	}

	return &header, nil
}
