package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
)

// C37ConfigurationFrame3 reprezentuje ramkę konfiguracji dla standardu C37.118.
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

// PhasorScaleFactor reprezentuje współczynnik konwersji dla kanałów fazorów z dodatkowymi flagami.
type PhasorScaleFactor struct {
	Flags           map[string]bool `json:"flags"`            // Flagi z mapowaniem bitowym
	PhasorType      string          `json:"phasor_type"`      // Typ fazora: Voltage lub Current
	PhasorComponent string          `json:"phasor_component"` // Komponent fazora (np. Phase A, Phase B)
	ScaleFactor     float32         `json:"scale_factor"`     // Współczynnik skali
	AngleOffset     float32         `json:"angle_offset"`     // Przesunięcie kąta
}

// AnalogScaleFactor reprezentuje współczynnik konwersji dla kanałów analogowych.
type AnalogScaleFactor struct {
	MagnitudeScale float32 `json:"magnitude_scale"` // Współczynnik skali wielkości w formacie IEEE 32-bit
	Offset         float32 `json:"offset"`          // Przesunięcie w formacie IEEE 32-bit
}

// DigitalMask reprezentuje maskę dla cyfrowych słów statusu.
type DigitalMask struct {
	Mask1 uint16 `json:"mask1"` // Pierwsza maska cyfrowa (16 bitów)
	Mask2 uint16 `json:"mask2"` // Druga maska cyfrowa (16 bitów)
}

// Definicje stałych dla bitu 0 w polu FORMAT
const (
	PhasorMagnitudeAndAngle = 0 // 0: magnitude i angle (polar)
	PhasorRealAndImaginary  = 1 // 1: real i imaginary (rectangular)
)

// FormatBits struktura reprezentująca bity pola FORMAT
type FormatBits struct {
	FREQ_DFREQ uint8 // Bit 3: Format częstotliwości DFREQ (0: 16-bit, 1: floating point)
	AnalogFmt  uint8 // Bit 2: Format analogowy (0: 16-bit, 1: floating point)
	PhasorFmt  uint8 // Bit 1: Format fazorów (0: 16-bit, 1: floating point)
	PhasorType uint8 // Bit 0: Typ fazora (0: magnitude i angle/polar, 1: real i imaginary/rectangular)
}

// Funkcja dekodująca bity pola FORMAT na strukturę FormatBits
func decodeFormatBits(format uint16) FormatBits {
	return FormatBits{
		FREQ_DFREQ: uint8((format >> 3) & 1), // Bit 3
		AnalogFmt:  uint8((format >> 2) & 1), // Bit 2
		PhasorFmt:  uint8((format >> 1) & 1), // Bit 1
		PhasorType: uint8(format & 1),        // Bit 0
	}
}

// TimeBaseBits struktura reprezentująca bity pola TIME_BASE
type TimeBaseBits struct {
	Reserved       uint32 // Bits 31-15: Zarezerwowane, zawsze 0
	TimeMultiplier uint32 // Bits 14-0: Mnożnik podstawy czasu
}

// Funkcja dekodująca bity pola TIME_BASE na strukturę TimeBaseBits
func decodeTimeBase(timeBase uint32) TimeBaseBits {
	return TimeBaseBits{
		Reserved:       (timeBase >> 15) & 0x1FFFF, // Bits 31-15
		TimeMultiplier: timeBase & 0x7FFF,          // Bits 14-0
	}
}

func decodeCHNAMWithOffsetAndLength(reader *bytes.Reader, totalNames int) ([]string, error) {
	channelNames := make([]string, totalNames)

	// Przesunięcie o 1 bajt
	if _, err := reader.Seek(-1, io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("error applying offset: %v", err)
	}

	for i := 0; i < totalNames; i++ {
		// Odczytaj długość nazwy (1 bajt)
		nameLen, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("error reading length of channel name at index %d: %v", i, err)
		}

		// Odczytaj nazwę o długości określonej w nameLen
		name := make([]byte, nameLen)
		if _, err := reader.Read(name); err != nil {
			return nil, fmt.Errorf("error reading channel name at index %d: %v", i, err)
		}

		// Konwertuj bajty na string i dodaj do listy nazw
		channelNames[i] = string(name)
	}

	return channelNames, nil
}

// TODO this can be used in Frame2
//func decodeCHNAMFixedLength(reader *bytes.Reader, totalNames int) ([]string, error) {
//	const nameLength = 16 // Każda nazwa ma stałe 16 bajtów
//	channelNames := make([]string, totalNames)
//
//	for i := 0; i < totalNames; i++ {
//		// Odczytaj nazwę o stałej długości
//		name := make([]byte, nameLength)
//		if _, err := reader.Read(name); err != nil {
//			return nil, fmt.Errorf("error reading channel name at index %d: %v", i, err)
//		}
//
//		// Trim trailing spaces or null bytes (jeśli są wypełnienia w nazwach)
//		channelNames[i] = string(bytes.TrimRight(name, "\x00 "))
//	}
//
//	return channelNames, nil
//}

// DecodeFlags dekoduje flagi na podstawie wartości uint16, zwracając mapę opisującą ustawione flagi
func DecodeFlags(flags uint16) map[string]bool {
	return map[string]bool{
		"reserved":                  (flags & 0x0001) != 0, // Bit 0: Zarezerwowane (nieużywane)
		"upsampled_with_interpol":   (flags & 0x0002) != 0, // Bit 1: Próbkowanie w górę za pomocą interpolacji
		"upsampled_with_extrapol":   (flags & 0x0004) != 0, // Bit 2: Próbkowanie w górę za pomocą ekstrapolacji
		"downsampled_with_reselect": (flags & 0x0008) != 0, // Bit 3: Próbkowanie w dół z wyborem próbek
		"downsampled_with_fir":      (flags & 0x0010) != 0, // Bit 4: Próbkowanie w dół z filtrem FIR
		"downsampled_non_fir":       (flags & 0x0020) != 0, // Bit 5: Próbkowanie w dół bez użycia filtra FIR
		"filtered_without_sampling": (flags & 0x0040) != 0, // Bit 6: Filtracja bez zmiany próbkowania
		"magnitude_adjusted":        (flags & 0x0080) != 0, // Bit 7: Dopasowanie wielkości
		"phase_adjusted_rotation":   (flags & 0x0100) != 0, // Bit 8: Dopasowanie fazy przez rotację
		"pseudo_phasor":             (flags & 0x0400) != 0, // Bit 10: Pseudofazor
		"modification_applied":      (flags & 0x8000) != 0, // Bit 15: Zastosowano modyfikację
	}
}

// DecodePhasorScale dekoduje PhasorScale z danych binarnych
func DecodePhasorScale(reader *bytes.Reader, count int) ([]PhasorScaleFactor, error) {
	phasorScales := make([]PhasorScaleFactor, count)

	for i := 0; i < count; i++ {
		var flags uint16
		var phasorTypeAndComponent uint8
		var reserved uint8
		var scaleFactor uint32
		var angleOffset uint32

		// Odczyt pierwszego 4-bajtowego słowa (flags + typ + komponent)
		if err := binary.Read(reader, binary.BigEndian, &flags); err != nil {
			return nil, fmt.Errorf("Błąd odczytu BitMappedFlags dla PhasorScale: %v", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &phasorTypeAndComponent); err != nil {
			return nil, fmt.Errorf("Błąd odczytu Typu i Komponentu dla PhasorScale: %v", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &reserved); err != nil {
			return nil, fmt.Errorf("Błąd odczytu Reserved dla PhasorScale: %v", err)
		}

		// Rozkodowanie flags
		decodedFlags := DecodeFlags(flags)

		// Rozbicie phasorTypeAndComponent na typ i komponent fazora
		phasorType := "voltage"
		if (phasorTypeAndComponent>>3)&0x01 == 1 {
			phasorType = "current"
		}

		phasorComponent := map[uint8]string{
			0b000: "zero sequence",
			0b001: "positive sequence",
			0b010: "negative sequence",
			0b011: "reserved",
			0b100: "phase A",
			0b101: "phase B",
			0b110: "phase C",
			0b111: "reserved",
		}[phasorTypeAndComponent&0x07]

		// Odczyt drugiego 4-bajtowego słowa - Scale Factor
		if err := binary.Read(reader, binary.BigEndian, &scaleFactor); err != nil {
			return nil, fmt.Errorf("Błąd odczytu ScaleFactor dla PhasorScale: %v", err)
		}

		// Odczyt trzeciego 4-bajtowego słowa - Angle Offset
		if err := binary.Read(reader, binary.BigEndian, &angleOffset); err != nil {
			return nil, fmt.Errorf("Błąd odczytu AngleOffset dla PhasorScale: %v", err)
		}

		// Konwersja ScaleFactor i AngleOffset do float32
		scaleFactorFloat := math.Float32frombits(scaleFactor)
		angleOffsetFloat := math.Float32frombits(angleOffset)

		phasorScales[i] = PhasorScaleFactor{
			Flags:           decodedFlags,
			PhasorType:      phasorType,
			PhasorComponent: phasorComponent,
			ScaleFactor:     scaleFactorFloat,
			AngleOffset:     angleOffsetFloat,
		}
	}
	return phasorScales, nil
}

// DecodeAnalogScale dekoduje AnalogScale z danych binarnych.
func DecodeAnalogScale(reader *bytes.Reader, count int) ([]AnalogScaleFactor, error) {
	analogScales := make([]AnalogScaleFactor, count)
	for i := 0; i < count; i++ {
		var scale AnalogScaleFactor
		if err := binary.Read(reader, binary.BigEndian, &scale.MagnitudeScale); err != nil {
			return nil, fmt.Errorf("Błąd odczytu MagnitudeScale dla AnalogScale: %v", err)
		}
		if err := binary.Read(reader, binary.BigEndian, &scale.Offset); err != nil {
			return nil, fmt.Errorf("Błąd odczytu Offset dla AnalogScale: %v", err)
		}
		analogScales[i] = scale
	}
	return analogScales, nil
}

// DecodeDigitalMask dekoduje maski cyfrowe z pola DIGUNIT o długości 4 bajtów.
func DecodeDigitalMask(reader *bytes.Reader, numDigitals uint16) ([]DigitalMask, error) {
	digitalMasks := make([]DigitalMask, numDigitals)

	numDigitals = 0 //TODO temporary hardcoded

	for i := 0; i < int(numDigitals); i++ {
		var mask DigitalMask

		// Pierwsze 2 bajty - maska 1
		if err := binary.Read(reader, binary.BigEndian, &mask.Mask1); err != nil {
			return nil, fmt.Errorf("Błąd odczytu Mask1 dla DigitalMask: %v", err)
		}

		// Kolejne 2 bajty - maska 2
		if err := binary.Read(reader, binary.BigEndian, &mask.Mask2); err != nil {
			return nil, fmt.Errorf("Błąd odczytu Mask2 dla DigitalMask: %v", err)
		}

		digitalMasks[i] = mask
	}

	return digitalMasks, nil
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
	header.TimeBase = decodeTimeBase(timeBase)

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
	channelNames, err := decodeCHNAMWithOffsetAndLength(reader, totalNames)
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
