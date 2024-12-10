package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// DataPhasor represents a phasor with real and imaginary components
type DataPhasor struct {
	Real      float32 // Real part of the phasor
	Imaginary float32 // Imaginary part of the phasor
}

// DataFields represents the data frame fields in C37.118
type DataFields struct {
	STAT     uint16       // Status word, 2 bytes
	Phasors  []DataPhasor // Array of phasors
	FREQ     float32      // Frequency, 4 bytes
	DFREQ    float32      // Rate of change of frequency, 4 bytes
	Analogs  []float32    // Array of analog values
	Digitals []uint16     // Array of digital status words
	CHK      uint16       // CRC for integrity check, 2 bytes
}

func DecodeDataFields(data []byte, phnmr, annmr, dgnmr int) (*DataFields, error) {
	// Wyświetlanie surowych danych w formacie hex dla diagnozy
	fmt.Printf("Raw data (hex): % X\n", data)

	// Sprawdź czy dane są odpowiedniej długości
	expectedLength := 2 + (phnmr * 8) + 4 + 4 + (annmr * 4) + (dgnmr * 2) + 2
	if len(data) < expectedLength {
		return nil, fmt.Errorf("not enough data for DataFields, expected at least %d bytes", expectedLength)
	}

	fields := &DataFields{}
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
