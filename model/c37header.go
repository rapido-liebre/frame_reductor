package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// FrameType określa typ ramki na podstawie bitów 6-4 w polu Sync
type FrameType int

const (
	DataFrame           FrameType = 0b000
	HeaderFrame         FrameType = 0b001
	ConfigurationFrame1 FrameType = 0b010
	ConfigurationFrame2 FrameType = 0b011
	ConfigurationFrame3 FrameType = 0b101
	CommandFrame        FrameType = 0b100
)

// Version określa wersję na podstawie bitów 3-0 w polu Sync
type Version int

const (
	Version1 Version = 0b0001 // C37118.2-2005
	Version2 Version = 0b0010 // C37118.2-2011
)

// C37Header zawiera podstawowe informacje nagłówka dla wszystkich ramek zgodnie z normą IEEE C37.118.2-2011
type C37Header struct {
	Sync      uint16 // Pole synchronizacji
	FrameSize uint16 // Rozmiar ramki
	IDCode    uint16 // Identyfikator urządzenia
	Soc       uint32 // Sekunda epoki Unix
	FracSec   uint32 // Część sekundy
	//Chk           uint16 // CRC-CCITT
	DataFrameType FrameType
	VersionNumber Version
	TimeStamp     time.Time
	FractionSec   FracSec
}

type FracSec struct {
	MessageTimeQuality uint8   // 8-bitowa jakość czasu
	FractionOfSecond   float64 // Ułamek sekundy jako float64
}

// DecodeC37Header dekoduje nagłówek ramki C37, analizując pola Sync, FRAMESIZE, IDCODE, Soc, Fracsec oraz Chk
func DecodeC37Header(data []byte) (*C37Header, error) {
	if len(data) < 14 {
		return nil, fmt.Errorf("długość danych %d jest zbyt krótka dla nagłówka", len(data))
	}

	reader := bytes.NewReader(data)
	header := &C37Header{}

	// Dekodowanie pól Sync, FrameSize, IDCode, Soc i FracSec
	if err := binary.Read(reader, binary.BigEndian, &header.Sync); err != nil {
		return nil, fmt.Errorf("błąd odczytu Sync: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.FrameSize); err != nil {
		return nil, fmt.Errorf("błąd odczytu FrameSize: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.IDCode); err != nil {
		return nil, fmt.Errorf("błąd odczytu IDCode: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.Soc); err != nil {
		return nil, fmt.Errorf("błąd odczytu Soc: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &header.FracSec); err != nil {
		return nil, fmt.Errorf("błąd odczytu Fracsec: %v", err)
	}

	//// Dekodowanie Chk
	//if err := binary.Read(reader, binary.BigEndian, &header.Chk); err != nil {
	//	return nil, fmt.Errorf("błąd odczytu Chk: %v", err)
	//}

	// Ustal typ ramki i wersję
	//fmt.Printf("Sync in hex: %X\n", header.Sync)
	header.DataFrameType = FrameType((header.Sync >> 4) & 0b111) // Bity 6-4
	header.VersionNumber = Version(header.Sync & 0b1111)         // Bity 3-0

	// Konwersja na czas
	unixTime := int64(header.Soc)                   // konwersja uint32 na int64
	header.TimeStamp = time.Unix(unixTime, 0).UTC() // tworzenie obiektu time.Time w UTC

	header.FractionSec = DecodeFracSec(header.FracSec, 1)

	return header, nil
}

func DecodeFracSec(fracSec uint32, timeBase uint32) FracSec {
	// Wyodrębnienie Message Time Quality (8 najwyższych bitów)
	messageTimeQuality := uint8(fracSec >> 24)

	// Wyodrębnienie Fractional Second (24 najniższych bitów)
	fractionalSecondRaw := fracSec & 0x00FFFFFF

	// Obliczenie rzeczywistego ułamka sekundy
	fractionOfSecond := float64(fractionalSecondRaw) / float64(timeBase)

	return FracSec{
		MessageTimeQuality: messageTimeQuality,
		FractionOfSecond:   fractionOfSecond,
	}
}

//// CalculateTimeUTC - funkcja obliczająca czas UTC na podstawie nagłówka
//func CalculateTimeUTC(header *C37Header) time.Time {
//	// Konwersja Soc na czas UTC
//	seconds := time.Unix(int64(header.Soc), 0).UTC()
//
//	// Obliczenie części sekundy
//	fracSeconds := float64(header.FracSec) / float64(header.TimeBase)
//	totalSeconds := seconds.Add(time.Duration(fracSeconds * float64(time.Second)))
//
//	return totalSeconds
//}
