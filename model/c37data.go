package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// C37DataFrame reprezentuje ramkę danych zdefiniowaną w standardzie C37.118-2011.
type C37DataFrame struct {
	Sync        uint16        `json:"sync"`         // Bajt synchronizujący i typ ramki
	FrameSize   uint16        `json:"frame_size"`   // Całkowity rozmiar ramki
	IDCode      uint16        `json:"id_code"`      // Identyfikator PMU/PDC
	SOC         uint32        `json:"soc"`          // Sekunda od epoki UNIX
	FracSec     FracSecBits   `json:"frac_sec"`     // Ułamek sekundy (FRACSEC)
	NumPhasors  uint16        `json:"num_phasors"`  // Liczba fazorów
	NumAnalogs  uint16        `json:"num_analogs"`  // Liczba wartości analogowych
	NumDigitals uint16        `json:"num_digitals"` // Liczba cyfrowych słów statusowych
	Phasors     []Phasor      `json:"phasors"`      // Dane fazorów
	Analogs     []float32     `json:"analogs"`      // Dane analogowe
	Digitals    []DigitalWord `json:"digitals"`     // Dane cyfrowe
	StatusFlags StatusFlags   `json:"status_flags"` // Flagi statusowe PMU
	CRC         uint16        `json:"crc"`          // Suma kontrolna CRC
}

// FracSecBits reprezentuje ułamek sekundy oraz status synchronizacji.
type FracSecBits struct {
	FractionOfSecond uint32 `json:"fraction_of_second"` // Ułamek sekundy
	TimeQuality      uint8  `json:"time_quality"`       // Bity jakości czasu
	Locked           bool   `json:"locked"`             // Synchronizacja z czasem
}

// Phasor reprezentuje dane fazora (wielkość i kąt lub składowe prostokątne).
type Phasor struct {
	Magnitude float32 `json:"magnitude"` // Wielkość fazora
	Angle     float32 `json:"angle"`     // Kąt fazora w stopniach
}

// DigitalWord reprezentuje cyfrowe słowo statusowe PMU.
type DigitalWord struct {
	Value uint16 `json:"value"` // Wartość słowa cyfrowego
}

// StatusFlags reprezentuje szczegółowe flagi statusowe PMU.
type StatusFlags struct {
	DataValid       bool `json:"data_valid"`       // Dane są poprawne
	PMUError        bool `json:"pmu_error"`        // Błąd PMU
	DataSorted      bool `json:"data_sorted"`      // Dane posortowane
	ConfigurationOK bool `json:"configuration_ok"` // Konfiguracja jest poprawna
}

func decodeFracSec(fracSec uint32) FracSecBits {
	return FracSecBits{
		FractionOfSecond: fracSec & 0xFFFFFF,
		TimeQuality:      uint8((fracSec >> 24) & 0x0F),
		Locked:           (fracSec>>28)&0x1 != 0,
	}
}

func decodeStatusFlags(flags uint16) StatusFlags {
	return StatusFlags{
		DataValid:       (flags & 0x0001) != 0,
		PMUError:        (flags & 0x0002) != 0,
		DataSorted:      (flags & 0x0004) != 0,
		ConfigurationOK: (flags & 0x0008) != 0,
	}
}

// DecodeDataFrame dekoduje ramkę danych C37.118.
func DecodeDataFrame(data []byte) (*C37DataFrame, error) {
	reader := bytes.NewReader(data)
	var frame C37DataFrame

	// Dekodowanie nagłówka
	if err := binary.Read(reader, binary.BigEndian, &frame.Sync); err != nil {
		return nil, fmt.Errorf("błąd odczytu Sync: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame.FrameSize); err != nil {
		return nil, fmt.Errorf("błąd odczytu FrameSize: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame.IDCode); err != nil {
		return nil, fmt.Errorf("błąd odczytu IDCode: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame.SOC); err != nil {
		return nil, fmt.Errorf("błąd odczytu SOC: %v", err)
	}

	// Dekodowanie FRACSEC
	var fracSec uint32
	if err := binary.Read(reader, binary.BigEndian, &fracSec); err != nil {
		return nil, fmt.Errorf("błąd odczytu FRACSEC: %v", err)
	}
	frame.FracSec = decodeFracSec(fracSec)

	// Liczba kanałów
	if err := binary.Read(reader, binary.BigEndian, &frame.NumPhasors); err != nil {
		return nil, fmt.Errorf("błąd odczytu NumPhasors: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame.NumAnalogs); err != nil {
		return nil, fmt.Errorf("błąd odczytu NumAnalogs: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &frame.NumDigitals); err != nil {
		return nil, fmt.Errorf("błąd odczytu NumDigitals: %v", err)
	}

	// Dekodowanie fazorów
	frame.Phasors = make([]Phasor, frame.NumPhasors)
	for i := 0; i < int(frame.NumPhasors); i++ {
		var magnitude, angle float32
		if err := binary.Read(reader, binary.BigEndian, &magnitude); err != nil {
			return nil, fmt.Errorf("błąd odczytu magnitude fazora: %v", err)
		}
		if err := binary.Read(reader, binary.BigEndian, &angle); err != nil {
			return nil, fmt.Errorf("błąd odczytu angle fazora: %v", err)
		}
		frame.Phasors[i] = Phasor{Magnitude: magnitude, Angle: angle}
	}

	// Dekodowanie analogów
	frame.Analogs = make([]float32, frame.NumAnalogs)
	for i := 0; i < int(frame.NumAnalogs); i++ {
		if err := binary.Read(reader, binary.BigEndian, &frame.Analogs[i]); err != nil {
			return nil, fmt.Errorf("błąd odczytu analogów: %v", err)
		}
	}

	// Dekodowanie cyfrowych słów
	frame.Digitals = make([]DigitalWord, frame.NumDigitals)
	for i := 0; i < int(frame.NumDigitals); i++ {
		var word uint16
		if err := binary.Read(reader, binary.BigEndian, &word); err != nil {
			return nil, fmt.Errorf("błąd odczytu DigitalWord: %v", err)
		}
		frame.Digitals[i] = DigitalWord{Value: word}
	}

	// Dekodowanie flag statusowych
	var status uint16
	if err := binary.Read(reader, binary.BigEndian, &status); err != nil {
		return nil, fmt.Errorf("błąd odczytu flag statusowych: %v", err)
	}
	frame.StatusFlags = decodeStatusFlags(status)

	// Dekodowanie CRC
	if err := binary.Read(reader, binary.BigEndian, &frame.CRC); err != nil {
		return nil, fmt.Errorf("błąd odczytu CRC: %v", err)
	}

	return &frame, nil
}

/*
// DataPhasor represents a phasor with real and imaginary components
type DataPhasor struct {
	Real      float32 // Real part of the phasor
	Imaginary float32 // Imaginary part of the phasor
}

// C37DataFrame represents the data frame fields in C37.118
type C37DataFrame struct {
	STAT     uint16       // Status word, 2 bytes
	Phasors  []DataPhasor // Array of phasors
	FREQ     float32      // Frequency, 4 bytes
	DFREQ    float32      // Rate of change of frequency, 4 bytes
	Analogs  []float32    // Array of analog values
	Digitals []uint16     // Array of digital status words
	CHK      uint16       // CRC for integrity check, 2 bytes
}

func DecodeDataFrame(data []byte , phnmr, annmr, dgnmr int) (*C37DataFrame, error) {
	// Wyświetlanie surowych danych w formacie hex dla diagnozy
	fmt.Printf("Raw data (hex): % X\n", data)

	// Sprawdź czy dane są odpowiedniej długości
	expectedLength := 2 + (phnmr * 8) + 4 + 4 + (annmr * 4) + (dgnmr * 2) + 2
	if len(data) < expectedLength {
		return nil, fmt.Errorf("not enough data for C37DataFrame, expected at least %d bytes", expectedLength)
	}

	fields := &C37DataFrame{}
	reader := bytes.NewReader(data)

	// Decode STAT (2 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.STAT); err != nil {
		return nil, fmt.Errorf("error decoding STAT: %v", err)
	}
	fmt.Printf("Decoded STAT: %v\n", fields.STAT)

	// Decode PHASORS (8 bytes per phasor: 4 bytes real, 4 bytes imaginary)
	fields.Phasors = make([]DataPhasor, phnmr)
	for i := 0; i < phnmr; i++ {
		if err := binary.Read(reader, binary.BigEndian, &fields.Phasors[i].Real); err != nil {
			return nil, fmt.Errorf("error decoding DataPhasor Real part: %v", err)
		}
		if err := binary.Read(reader, binary.BigEndian, &fields.Phasors[i].Imaginary); err != nil {
			return nil, fmt.Errorf("error decoding DataPhasor Imaginary part: %v", err)
		}
		fmt.Printf("Decoded DataPhasor %d: Real=%v, Imaginary=%v\n", i, fields.Phasors[i].Real, fields.Phasors[i].Imaginary)
	}

	// Decode FREQ (4 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.FREQ); err != nil {
		return nil, fmt.Errorf("error decoding FREQ: %v", err)
	}
	fmt.Printf("Decoded FREQ: %v\n", fields.FREQ)

	// Decode DFREQ (4 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.DFREQ); err != nil {
		return nil, fmt.Errorf("error decoding DFREQ: %v", err)
	}
	fmt.Printf("Decoded DFREQ: %v\n", fields.DFREQ)

	// Decode ANALOG (4 bytes per analog value)
	fields.Analogs = make([]float32, annmr)
	for i := 0; i < annmr; i++ {
		if err := binary.Read(reader, binary.BigEndian, &fields.Analogs[i]); err != nil {
			return nil, fmt.Errorf("error decoding Analog value: %v", err)
		}
		fmt.Printf("Decoded Analog %d: %v\n", i, fields.Analogs[i])
	}

	// Decode DIGITAL (2 bytes per digital status word)
	fields.Digitals = make([]uint16, dgnmr)
	for i := 0; i < dgnmr; i++ {
		if err := binary.Read(reader, binary.BigEndian, &fields.Digitals[i]); err != nil {
			return nil, fmt.Errorf("error decoding Digital status: %v", err)
		}
		fmt.Printf("Decoded Digital %d: %v\n", i, fields.Digitals[i])
	}

	// Decode Chk (2 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.CHK); err != nil {
		return nil, fmt.Errorf("error decoding Chk: %v", err)
	}
	fmt.Printf("Decoded Chk: %v\n", fields.CHK)

	return fields, nil
}
*/
