package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// C37Header - struktura nagłówka IEEE C37.118
type C37Header struct {
	IDCode    uint16 // Identyfikator urządzenia
	FrameSize uint16 // Rozmiar ramki
	FrameType uint16 // Typ ramki
	Soc       uint32 // Sekunda epoki Unix
	FracSec   uint32 // Część sekundy
	TimeBase  uint32 // Podstawa czasu
}

// DataFields defines the structure for data after the header in C37.118 frame
type DataFields struct {
	STAT  uint16  // Status word, 2 bytes
	FREQ  float32 // Frequency, 4 bytes
	DFREQ float32 // Rate of change of frequency, 4 bytes
	CHK   uint16  // CRC for integrity check, 2 bytes
	// Other fields like PHASORS, ANALOG, and DIGITAL are placeholders
}

// DecodeC37Header - funkcja do dekodowania nagłówka
func DecodeC37Header(data []byte) (*C37Header, error) {
	//fmt.Printf("Nagłówek ma %d bajtów\n", len(data))
	if len(data) < 18 {
		return nil, fmt.Errorf("za mało danych na nagłówek, wymagane 16 bajtów, otrzymano %d", len(data))
	}

	header := &C37Header{}
	reader := bytes.NewReader(data[:18])
	if err := binary.Read(reader, binary.BigEndian, header); err != nil {
		return nil, fmt.Errorf("błąd podczas dekodowania nagłówka: %v", err)
	}
	return header, nil
}

// CalculateTimeUTC - funkcja obliczająca czas UTC na podstawie nagłówka
func CalculateTimeUTC(header *C37Header) time.Time {
	// Konwersja SOC na czas UTC
	seconds := time.Unix(int64(header.Soc), 0).UTC()

	// Obliczenie części sekundy
	fracSeconds := float64(header.FracSec) / float64(header.TimeBase)
	totalSeconds := seconds.Add(time.Duration(fracSeconds * float64(time.Second)))

	return totalSeconds
}

// DecodeDataFields decodes the data fields from the remaining bytes of the frame
func DecodeDataFields(data []byte) (*DataFields, error) {
	if len(data) < 12 { // STAT(2) + FREQ(4) + DFREQ(4) + CHK(2) = 12 bytes
		return nil, fmt.Errorf("not enough data for DataFields, expected at least 12 bytes")
	}

	fields := &DataFields{}
	reader := bytes.NewReader(data[:12])

	// Decode STAT (2 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.STAT); err != nil {
		return nil, fmt.Errorf("error decoding STAT: %v", err)
	}

	// Decode FREQ (4 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.FREQ); err != nil {
		return nil, fmt.Errorf("error decoding FREQ: %v", err)
	}

	// Decode DFREQ (4 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.DFREQ); err != nil {
		return nil, fmt.Errorf("error decoding DFREQ: %v", err)
	}

	// Decode CHK (2 bytes)
	if err := binary.Read(reader, binary.BigEndian, &fields.CHK); err != nil {
		return nil, fmt.Errorf("error decoding CHK: %v", err)
	}

	return fields, nil
}
