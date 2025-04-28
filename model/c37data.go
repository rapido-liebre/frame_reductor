package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// C37DataFrame reprezentuje ramkę danych zdefiniowaną w standardzie C37.118
type C37DataFrame struct {
	C37Header
	Stat      Stat      `json:"stat"`      // Flagi bitowe
	Phasors   []Phasor  `json:"phasors"`   // Dane fazorów
	Frequency float64   `json:"frequency"` // Dane częstotliwości
	Rocof     float64   `json:"rocof"`     // Dane dF/dt ROCOF
	Analogs   []Analog  `json:"analogs"`   // Dane analogowe
	Digitals  []Digital `json:"digitals"`  // Dane cyfrowe
	CRC       uint16    `json:"crc"`       // Suma kontrolna CRC
}

// Stat reprezentuje zdekodowane wartości pól STAT w ramce C37.118
type Stat struct {
	DataError      string // Bity 15–14: Błąd danych
	PMUSync        bool   // Bit 13: PMU zsynchronizowany
	DataSorting    bool   // Bit 12: Sortowanie danych
	PMUTrigger     bool   // Bit 11: Wykryto wyzwalacz PMU
	ConfigChange   bool   // Bit 10: Zmiana konfiguracji
	DataModified   bool   // Bit 09: Dane zmodyfikowane
	PMUTimeQuality string // Bity 08–06: Jakość czasu PMU
	UnlockedTime   string // Bity 05–04: Czas odblokowania synchronizacji
	TriggerReason  string // Bity 03–00: Powód wyzwalacza
}

// Phasor reprezentuje dane pojedynczego fazora (wielkość i kąt lub składowe prostokątne)
type Phasor struct {
	Name      string      // Nazwa fazora
	Type      ChannelType // Typ fazora (napięcie/prąd)
	Magnitude float64     // Wielkość fazora (polar) lub część rzeczywista (rectangular)
	Angle     float64     // Kąt fazora w radianach (polar) lub część urojona (rectangular)
}

// Analog reprezentuje odczytaną wartość kanału analogowego
type Analog struct {
	Name  string  // Nazwa analogu
	Value float64 // Wartość analogowa
}

// Digital reprezentuje zdekodowany kanał cyfrowy, słowo statusowe PMU
type Digital struct {
	Name  string // Nazwa kanału cyfrowego
	Value bool   // Wartość cyfrowa (true/false)
}

// StatusFlags reprezentuje szczegółowe flagi statusowe PMU
type StatusFlags struct {
	DataValid       bool `json:"data_valid"`       // Dane są poprawne
	PMUError        bool `json:"pmu_error"`        // Błąd PMU
	DataSorted      bool `json:"data_sorted"`      // Dane posortowane
	ConfigurationOK bool `json:"configuration_ok"` // Konfiguracja jest poprawna
}

// DecodeDataFrame dekoduje ramkę danych C37.118
func DecodeDataFrame(data []byte, header C37Header) (*C37DataFrame, error) {
	reader := bytes.NewReader(data)
	var frame C37DataFrame

	frame.C37Header = header

	// Dekodowanie flag bitowych
	var stat uint16
	if err := binary.Read(reader, binary.BigEndian, &stat); err != nil {
		return nil, fmt.Errorf("błąd odczytu flag bitowych Stat: %v", err)
	}
	frame.Stat = DecodeStat(stat)

	// Liczba kanałów
	format := CfgFrame2.Format

	// Dekodowanie fazorów
	phasors, err := DecodePhasors(reader, format)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu fazorów: %v", err)
	}
	frame.Phasors = phasors

	// Dekodowanie częstotliwości
	freq, err := DecodeFrequency(reader, format)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu częstotliwości: %v", err)
	}
	frame.Frequency = freq

	// Dekodowanie dF/dt
	rocof, err := DecodeROCOF(reader, format)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu ROCOF dF/dt: %v", err)
	}
	frame.Rocof = rocof

	analogs, err := DecodeAnalogs(reader, format)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu kanałów analogowych: %v", err)
	}
	frame.Analogs = analogs

	digitals, err := DecodeDigitals(reader)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu kanałów cyfrowych: %v", err)
	}
	frame.Digitals = digitals

	// Dekodowanie CRC
	if err := binary.Read(reader, binary.BigEndian, &frame.CRC); err != nil {
		return nil, fmt.Errorf("błąd odczytu CRC: %v", err)
	}

	return &frame, nil
}

// DecodeStat dekoduje wartość STAT (16-bitową mapę bitów) na strukturę Stat, zawierającą szczegółowe informacje
// o stanie PMU, takie jak błędy danych, jakość czasu, powód wyzwalacza i inne flagi.
func DecodeStat(stat uint16) Stat {
	dataErrorMap := map[uint8]string{
		0b00: "Dobre dane pomiarowe, brak błędów",
		0b01: "Błąd PMU. Brak informacji o danych",
		0b10: "PMU w trybie testowym lub brak danych",
		0b11: "Błąd PMU (nie używać wartości)",
	}

	unlockedTimeMap := map[uint8]string{
		0b00: "Synchronizacja zablokowana lub odblokowana < 10 s (najlepsza jakość)",
		0b01: "10 s ≤ odblokowany czas < 100 s",
		0b10: "100 s < odblokowany czas ≤ 1000 s",
		0b11: "Odblokowany czas > 1000 s",
	}

	triggerReasonMap := map[uint8]string{
		0b1111: "Definicja użytkownika",
		0b0111: "Cyfrowe",
		0b0101: "df/dt wysokie",
		0b0011: "Różnica kąta fazowego",
		0b0001: "Mała amplituda",
		0b0110: "Zarezerwowane",
		0b0100: "Wysoka lub niska częstotliwość",
		0b0010: "Duża amplituda",
		0b0000: "Manualne",
	}

	pmuTimeQualityMap := map[uint8]string{
		0b111: "Maksymalny błąd czasu > 10 ms lub nieznany",
		0b110: "Maksymalny błąd czasu < 10 ms",
		0b101: "Maksymalny błąd czasu < 1 ms",
		0b100: "Maksymalny błąd czasu < 100 μs",
		0b011: "Maksymalny błąd czasu < 10 μs",
		0b010: "Maksymalny błąd czasu < 1 μs",
		0b001: "Maksymalny błąd czasu < 100 ns",
		0b000: "Nie używany (kod z poprzedniej wersji profilu)",
	}

	// Dekodowanie bitów
	dataError := uint8((stat >> 14) & 0b11)
	pmuSync := (stat>>13)&1 == 0
	dataSorting := (stat>>12)&1 == 1
	pmuTrigger := (stat>>11)&1 == 1
	configChange := (stat>>10)&1 == 1
	dataModified := (stat>>9)&1 == 1
	pmuTimeQuality := uint8((stat >> 6) & 0b111)
	unlockedTime := uint8((stat >> 4) & 0b11)
	triggerReason := uint8(stat & 0b1111)

	return Stat{
		DataError:      dataErrorMap[dataError],
		PMUSync:        pmuSync,
		DataSorting:    dataSorting,
		PMUTrigger:     pmuTrigger,
		ConfigChange:   configChange,
		DataModified:   dataModified,
		PMUTimeQuality: pmuTimeQualityMap[pmuTimeQuality],
		UnlockedTime:   unlockedTimeMap[unlockedTime],
		TriggerReason:  triggerReasonMap[triggerReason],
	}
}

// EncodeStat koduje strukturę Stat na 16-bitową wartość STAT.
func EncodeStat(stat Stat) (uint16, error) {
	// Mapa odwrotna do dekodowania DataError
	dataErrorMap := map[string]uint8{
		"Dobre dane pomiarowe, brak błędów":     0b00,
		"Błąd PMU. Brak informacji o danych":    0b01,
		"PMU w trybie testowym lub brak danych": 0b10,
		"Błąd PMU (nie używać wartości)":        0b11,
	}

	// Mapa odwrotna do dekodowania UnlockedTime
	unlockedTimeMap := map[string]uint8{
		"Synchronizacja zablokowana lub odblokowana < 10 s (najlepsza jakość)": 0b00,
		"10 s ≤ odblokowany czas < 100 s":                                      0b01,
		"100 s < odblokowany czas ≤ 1000 s":                                    0b10,
		"Odblokowany czas > 1000 s":                                            0b11,
	}

	// Mapa odwrotna do dekodowania TriggerReason
	triggerReasonMap := map[string]uint8{
		"Definicja użytkownika":          0b1111,
		"Cyfrowe":                        0b0111,
		"df/dt wysokie":                  0b0101,
		"Różnica kąta fazowego":          0b0011,
		"Mała amplituda":                 0b0001,
		"Zarezerwowane":                  0b0110,
		"Wysoka lub niska częstotliwość": 0b0100,
		"Duża amplituda":                 0b0010,
		"Manualne":                       0b0000,
	}

	// Mapa odwrotna do dekodowania PMUTimeQuality
	pmuTimeQualityMap := map[string]uint8{
		"Maksymalny błąd czasu > 10 ms lub nieznany":     0b111,
		"Maksymalny błąd czasu < 10 ms":                  0b110,
		"Maksymalny błąd czasu < 1 ms":                   0b101,
		"Maksymalny błąd czasu < 100 μs":                 0b100,
		"Maksymalny błąd czasu < 10 μs":                  0b011,
		"Maksymalny błąd czasu < 1 μs":                   0b010,
		"Maksymalny błąd czasu < 100 ns":                 0b001,
		"Nie używany (kod z poprzedniej wersji profilu)": 0b000,
	}

	// Kodowanie poszczególnych pól
	var encoded uint16

	// DataError
	dataError, ok := dataErrorMap[stat.DataError]
	if !ok {
		return 0, fmt.Errorf("nieprawidłowa wartość DataError: %s", stat.DataError)
	}
	encoded |= uint16(dataError) << 14

	// PMUSync
	if !stat.PMUSync {
		encoded |= 1 << 13
	}

	// DataSorting
	if stat.DataSorting {
		encoded |= 1 << 12
	}

	// PMUTrigger
	if stat.PMUTrigger {
		encoded |= 1 << 11
	}

	// ConfigChange
	if stat.ConfigChange {
		encoded |= 1 << 10
	}

	// DataModified
	if stat.DataModified {
		encoded |= 1 << 9
	}

	// PMUTimeQuality
	pmuTimeQuality, ok := pmuTimeQualityMap[stat.PMUTimeQuality]
	if !ok {
		return 0, fmt.Errorf("nieprawidłowa wartość PMUTimeQuality: %s", stat.PMUTimeQuality)
	}
	encoded |= uint16(pmuTimeQuality) << 6

	// UnlockedTime
	unlockedTime, ok := unlockedTimeMap[stat.UnlockedTime]
	if !ok {
		return 0, fmt.Errorf("nieprawidłowa wartość UnlockedTime: %s", stat.UnlockedTime)
	}
	encoded |= uint16(unlockedTime) << 4

	// TriggerReason
	triggerReason, ok := triggerReasonMap[stat.TriggerReason]
	if !ok {
		return 0, fmt.Errorf("nieprawidłowa wartość TriggerReason: %s", stat.TriggerReason)
	}
	encoded |= uint16(triggerReason)

	return encoded, nil
}

// DecodePhasors dekoduje fazory (PHASORS) na podstawie konfiguracji i formatu
func DecodePhasors(reader *bytes.Reader, format FormatBits) ([]Phasor, error) {
	phasors := make([]Phasor, len(CfgFrame2.PhasorUnits))

	for i := 0; i < len(CfgFrame2.PhasorUnits); i++ {
		var magnitude float64
		var angle float64

		if format.PhasorFmt == 0 { // 16-bit format
			if format.PhasorType == 0 { // Polar format
				var rawMagnitude uint16
				var rawAngle int16

				if err := binary.Read(reader, binary.BigEndian, &rawMagnitude); err != nil {
					return nil, fmt.Errorf("błąd odczytu wielkości fazora (polar, 16-bit): %v", err)
				}
				if err := binary.Read(reader, binary.BigEndian, &rawAngle); err != nil {
					return nil, fmt.Errorf("błąd odczytu kąta fazora (polar, 16-bit): %v", err)
				}

				magnitude = float64(rawMagnitude)
				angle = float64(rawAngle) / 10000.0 // Skala w radianach
			} else { // Rectangular format
				var realValue int16
				var imaginaryValue int16

				if err := binary.Read(reader, binary.BigEndian, &realValue); err != nil {
					return nil, fmt.Errorf("błąd odczytu części rzeczywistej fazora (rectangular, 16-bit): %v", err)
				}
				if err := binary.Read(reader, binary.BigEndian, &imaginaryValue); err != nil {
					return nil, fmt.Errorf("błąd odczytu części urojonej fazora (rectangular, 16-bit): %v", err)
				}

				magnitude = float64(realValue)
				angle = float64(imaginaryValue)
			}
		} else { // Floating point (32-bit) format
			if format.PhasorType == 0 { // Rectangular format
				var realVal float32
				var imaginaryVal float32

				if err := binary.Read(reader, binary.BigEndian, &realVal); err != nil {
					return nil, fmt.Errorf("błąd odczytu części rzeczywistej fazora (rectangular, floating point): %v", err)
				}
				if err := binary.Read(reader, binary.BigEndian, &imaginaryVal); err != nil {
					return nil, fmt.Errorf("błąd odczytu części urojonej fazora (rectangular, floating point): %v", err)
				}

				magnitude = float64(realVal)
				angle = float64(imaginaryVal)
			} else { // Polar format
				var rawMagnitude float32
				var rawAngle float32

				if err := binary.Read(reader, binary.BigEndian, &rawMagnitude); err != nil {
					return nil, fmt.Errorf("błąd odczytu wielkości fazora (polar, floating point): %v", err)
				}
				if err := binary.Read(reader, binary.BigEndian, &rawAngle); err != nil {
					return nil, fmt.Errorf("błąd odczytu kąta fazora (polar, floating point): %v", err)
				}

				magnitude = float64(rawMagnitude)
				angle = float64(rawAngle)
			}
		}

		phasors[i] = Phasor{
			Name:      strings.TrimRight(CfgFrame2.ChannelNames[i], "\x00"), // Nazwa z konfiguracji
			Type:      CfgFrame2.PhasorUnits[i].ChannelType,                 // Typ kanału (napięcie/prąd)
			Magnitude: magnitude,
			Angle:     angle,
		}
	}

	return phasors, nil
}

// EncodePhasors koduje fazory i zwraca ich bajty.
// Funkcja pozostawia tylko fazor U_SEQ+ i usuwa pozostałe.
func EncodePhasors(phasors []Phasor) ([]byte, error) {
	var buf bytes.Buffer
	format := CfgFrame2.Format

	// Znajdź fazor "U_SEQ+"
	var selectedPhasor *Phasor
	for _, phasor := range phasors {
		if strings.Contains(phasor.Name, "U_SEQ+") || strings.Contains(phasor.Name, "zgodna U") {
			selectedPhasor = &phasor
			break
		}
	}

	if selectedPhasor == nil {
		return nil, fmt.Errorf("fazor U_SEQ+ nie został znaleziony")
	}

	// Kodowanie fazora "U_SEQ+"
	if format.PhasorFmt == 0 { // 16-bit format
		if format.PhasorType == 0 { // Polar format
			// Zapis wielkości
			rawMagnitude := uint16(selectedPhasor.Magnitude)
			if err := binary.Write(&buf, binary.BigEndian, rawMagnitude); err != nil {
				return nil, fmt.Errorf("błąd zapisu wielkości fazora (polar, 16-bit): %v", err)
			}

			// Zapis kąta
			rawAngle := int16(selectedPhasor.Angle * 10000.0) // Skala w radianach
			if err := binary.Write(&buf, binary.BigEndian, rawAngle); err != nil {
				return nil, fmt.Errorf("błąd zapisu kąta fazora (polar, 16-bit): %v", err)
			}
		} else { // Rectangular format
			// Zapis części rzeczywistej
			realValue := int16(selectedPhasor.Magnitude)
			if err := binary.Write(&buf, binary.BigEndian, realValue); err != nil {
				return nil, fmt.Errorf("błąd zapisu części rzeczywistej fazora (rectangular, 16-bit): %v", err)
			}

			// Zapis części urojonej
			imaginaryValue := int16(selectedPhasor.Angle)
			if err := binary.Write(&buf, binary.BigEndian, imaginaryValue); err != nil {
				return nil, fmt.Errorf("błąd zapisu części urojonej fazora (rectangular, 16-bit): %v", err)
			}
		}
	} else { // Floating point (32-bit) format
		if format.PhasorType == 0 { // Polar format
			// Zapis wielkości
			rawMagnitude := float32(selectedPhasor.Magnitude)
			if err := binary.Write(&buf, binary.BigEndian, rawMagnitude); err != nil {
				return nil, fmt.Errorf("błąd zapisu wielkości fazora (polar, floating point): %v", err)
			}

			// Zapis kąta
			rawAngle := float32(selectedPhasor.Angle)
			if err := binary.Write(&buf, binary.BigEndian, rawAngle); err != nil {
				return nil, fmt.Errorf("błąd zapisu kąta fazora (polar, floating point): %v", err)
			}
		} else { // Rectangular format
			// Zapis części rzeczywistej
			realValue := float32(selectedPhasor.Magnitude)
			if err := binary.Write(&buf, binary.BigEndian, realValue); err != nil {
				return nil, fmt.Errorf("błąd zapisu części rzeczywistej fazora (rectangular, floating point): %v", err)
			}

			// Zapis części urojonej
			imaginaryValue := float32(selectedPhasor.Angle)
			if err := binary.Write(&buf, binary.BigEndian, imaginaryValue); err != nil {
				return nil, fmt.Errorf("błąd zapisu części urojonej fazora (rectangular, floating point): %v", err)
			}
		}
	}

	return buf.Bytes(), nil
}

// DecodeFrequency dekoduje częstotliwość (FREQ) na podstawie konfiguracji i formatu
func DecodeFrequency(reader *bytes.Reader, format FormatBits) (float64, error) {
	var frequency float64
	var nominalFrequency float64

	// Odczytaj nominalną częstotliwość ze struktury FNom
	if CfgFrame2.FNom.Is50Hz {
		nominalFrequency = 50.0 // 50 Hz
	} else if CfgFrame2.FNom.Is60Hz {
		nominalFrequency = 60.0 // 60 Hz
	} else {
		return 0, fmt.Errorf("nieznana nominalna częstotliwość: FNom nie wskazuje ani 50 Hz, ani 60 Hz")
	}

	// Dekodowanie na podstawie formatu
	if format.FREQ_DFREQ == 0 { // 16-bit integer format
		var rawFreq int16
		if err := binary.Read(reader, binary.BigEndian, &rawFreq); err != nil {
			return 0, fmt.Errorf("błąd odczytu częstotliwości (16-bit): %v", err)
		}

		// Przeskaluj wartość na mHz i dodaj do nominalnej częstotliwości
		frequency = nominalFrequency + float64(rawFreq)/1000.0
	} else { // 32-bit floating-point format
		// Odczytaj surowe bajty przed dekodowaniem
		rawBytes := make([]byte, 4) // 4 bajty dla float32
		if _, err := reader.Read(rawBytes); err != nil {
			return 0, fmt.Errorf("błąd odczytu surowych bajtów częstotliwości: %v", err)
		}

		// Dekoduj bajty jako float32
		var rawFreq float32
		if err := binary.Read(bytes.NewReader(rawBytes), binary.BigEndian, &rawFreq); err != nil {
			return 0, fmt.Errorf("błąd odczytu częstotliwości (floating-point): %v", err)
		}

		// Wartość surowa już zawiera pełną częstotliwość
		frequency = float64(rawFreq)
	}

	return frequency, nil
}

//// EncodeFrequency koduje wartość częstotliwości (FREQ) na podstawie konfiguracji i formatu.
//// Funkcja zwraca zakodowane bajty reprezentujące częstotliwość.
//func EncodeFrequency(frequency float64) ([]byte, error) {
//	var buf bytes.Buffer
//	var nominalFrequency float64
//	format := CfgFrame2.Format
//
//	// Odczytaj nominalną częstotliwość ze struktury FNom
//	if CfgFrame2.FNom.Is50Hz {
//		nominalFrequency = 50.0 // 50 Hz
//	} else if CfgFrame2.FNom.Is60Hz {
//		nominalFrequency = 60.0 // 60 Hz
//	} else {
//		return nil, fmt.Errorf("nieznana nominalna częstotliwość: FNom nie wskazuje ani 50 Hz, ani 60 Hz")
//	}
//
//	// Kodowanie na podstawie formatu
//	if format.FREQ_DFREQ == 0 { // 16-bit integer format
//		// Oblicz różnicę w stosunku do nominalnej częstotliwości i przeskaluj na mHz
//		rawFreq := int16((frequency - nominalFrequency) * 1000.0)
//
//		// Zapisz wartość jako 16-bitowy integer
//		if err := binary.Write(&buf, binary.BigEndian, rawFreq); err != nil {
//			return nil, fmt.Errorf("błąd zapisu częstotliwości (16-bit): %v", err)
//		}
//	} else { // 32-bit floating-point format
//		// Zapisz wartość częstotliwości jako 32-bitowy float
//		rawFreq := float32(frequency)
//		if err := binary.Write(&buf, binary.BigEndian, rawFreq); err != nil {
//			return nil, fmt.Errorf("błąd zapisu częstotliwości (floating-point): %v", err)
//		}
//	}
//
//	return buf.Bytes(), nil
//}

// EncodeFrequency koduje wartość częstotliwości (FREQ) na podstawie konfiguracji i formatu.
func EncodeFrequency(frequency float64) ([]byte, error) {
	var buf bytes.Buffer

	// Jeśli chcesz zawsze kodować jako float32 (zmiennoprzecinkowa)
	rawFreq := float32(frequency)

	// Zapisz wartość częstotliwości jako 32-bitowy float (IEEE 754)
	if err := binary.Write(&buf, binary.BigEndian, rawFreq); err != nil {
		return nil, fmt.Errorf("błąd zapisu częstotliwości (floating-point): %v", err)
	}

	return buf.Bytes(), nil
}

// DecodeROCOF dekoduje dF/dt (DFREQ) na podstawie konfiguracji i formatu
func DecodeROCOF(reader *bytes.Reader, format FormatBits) (float64, error) {
	var dfreq float64

	if format.FREQ_DFREQ == 0 { // 16-bit integer format
		// Odczytujemy surowe bajty (2 bajty)
		rawBytes := make([]byte, 2)
		if _, err := reader.Read(rawBytes); err != nil {
			return 0, fmt.Errorf("błąd odczytu DFREQ (16-bit): %v", err)
		}

		// Konwersja na wartość 16-bitową
		rawDFREQ := int16(rawBytes[0])<<8 | int16(rawBytes[1])
		//fmt.Printf("Odczytane bajty (16-bit): %X %X, rawDFREQ: %d\n", rawBytes[0], rawBytes[1], rawDFREQ)

		// Przeskalowanie na Hz/s (x100)
		dfreq = float64(rawDFREQ) / 100.0
	} else { // 32-bit floating-point format
		// Odczytujemy surowe bajty (4 bajty)
		rawBytes := make([]byte, 4)
		if _, err := reader.Read(rawBytes); err != nil {
			return 0, fmt.Errorf("błąd odczytu DFREQ (floating-point): %v", err)
		}

		// Konwersja na wartość 32-bitową zmiennoprzecinkową
		rawDFREQ := math.Float32frombits(binary.BigEndian.Uint32(rawBytes))
		//fmt.Printf("Odczytane bajty (32-bit): %X %X %X %X, rawDFREQ: %.3f\n", rawBytes[0], rawBytes[1], rawBytes[2], rawBytes[3], rawDFREQ)

		// Wartość jest już w Hz/s
		dfreq = float64(rawDFREQ)
	}

	return dfreq, nil
}

// EncodeROCOF koduje wartość ROCOF (DFREQ) na podstawie konfiguracji i formatu.
// Funkcja zwraca zakodowane bajty reprezentujące ROCOF.
func EncodeROCOF(dfreq float64) ([]byte, error) {
	var buf bytes.Buffer
	format := CfgFrame2.Format

	// Kodowanie na podstawie formatu
	if format.FREQ_DFREQ == 0 { // 16-bit integer format
		// Przeskalowanie wartości do formatu 16-bitowego (x100)
		rawDFREQ := int16(dfreq * 100.0)

		// Zapisz wartość jako 16-bitowy integer
		if err := binary.Write(&buf, binary.BigEndian, rawDFREQ); err != nil {
			return nil, fmt.Errorf("błąd zapisu DFREQ (16-bit): %v", err)
		}
	} else { // 32-bit floating-point format
		// Przekształcenie wartości na 32-bitowy float
		rawDFREQ := float32(dfreq)

		// Zapisz wartość jako 32-bitowy float
		if err := binary.Write(&buf, binary.BigEndian, rawDFREQ); err != nil {
			return nil, fmt.Errorf("błąd zapisu DFREQ (floating-point): %v", err)
		}
	}

	return buf.Bytes(), nil
}

// DecodeAnalogs dekoduje analogi (ANALOG) na podstawie konfiguracji i formatu
func DecodeAnalogs(reader *bytes.Reader, format FormatBits) ([]Analog, error) {
	// Liczba analogów
	numAnalogs := int(CfgFrame2.NumAnalogs)

	// Pobranie nazw analogów z konfiguracji (po nazwach fazorów)
	analogNames := CfgFrame2.ChannelNames[CfgFrame2.NumPhasors : CfgFrame2.NumPhasors+uint16(numAnalogs)]

	// Walidacja liczby analogów, jednostek i nazw
	if uint16(len(CfgFrame2.AnalogUnits)) != CfgFrame2.NumAnalogs || len(analogNames) != numAnalogs {
		return nil, fmt.Errorf(
			"niezgodność liczby analogów, jednostek lub nazw (numAnalogs: %d, len(analogUnits): %d, len(analogNames): %d)",
			numAnalogs, len(CfgFrame2.AnalogUnits), len(analogNames),
		)
	}

	// Dekodowanie analogów
	analogs := make([]Analog, numAnalogs)
	for i := 0; i < numAnalogs; i++ {
		unit := CfgFrame2.AnalogUnits[i]
		name := analogNames[i]
		var value float64

		// Dekodowanie wartości na podstawie formatu danych
		if format.AnalogFmt == 0 {
			// 16-bit integer
			var rawValue int16
			if err := binary.Read(reader, binary.BigEndian, &rawValue); err != nil {
				return nil, fmt.Errorf("błąd odczytu 16-bitowej wartości analogowej: %v", err)
			}
			value = float64(rawValue) * unit.ScalingFactor
		} else {
			// 32-bit floating-point
			var rawValue float32
			if err := binary.Read(reader, binary.BigEndian, &rawValue); err != nil {
				return nil, fmt.Errorf("błąd odczytu 32-bitowej wartości analogowej: %v", err)
			}
			value = float64(rawValue)
		}

		// Tworzenie struktury Analog
		analogs[i] = Analog{
			Name:  name,
			Value: value,
		}
	}

	return analogs, nil
}

// DecodeDigitals dekoduje dane cyfrowe na podstawie konfiguracji ramki
func DecodeDigitals(reader *bytes.Reader) ([]Digital, error) {
	// Liczba słów cyfrowych w konfiguracji
	numDigitalWords := int(CfgFrame2.NumDigitals)
	digitalNames := CfgFrame2.ChannelNames[CfgFrame2.NumPhasors+CfgFrame2.NumAnalogs:] // Nazwy cyfrowe zaczynają się po fazorach i analogach

	if len(digitalNames) != numDigitalWords*16 {
		return nil, fmt.Errorf(
			"niezgodność liczby nazw cyfrowych (numDigitalWords: %d, len(digitalNames): %d)",
			numDigitalWords, len(digitalNames),
		)
	}

	// Przechowywanie zdekodowanych kanałów cyfrowych
	digitals := []Digital{}

	// Dekodowanie każdego słowa cyfrowego
	for wordIndex := 0; wordIndex < numDigitalWords; wordIndex++ {
		// Odczyt 16-bitowego słowa cyfrowego
		var digitalWord uint16
		if err := binary.Read(reader, binary.BigEndian, &digitalWord); err != nil {
			return nil, fmt.Errorf("błąd odczytu cyfrowego słowa: %v", err)
		}

		// Dekodowanie bitów w słowie cyfrowym
		for bitIndex := 0; bitIndex < 16; bitIndex++ {
			bitValue := (digitalWord & (1 << bitIndex)) != 0 // Sprawdzenie, czy bit jest ustawiony

			// Pobranie nazwy kanału cyfrowego
			nameIndex := wordIndex*16 + bitIndex
			name := digitalNames[nameIndex]

			// Dodanie zdekodowanego kanału cyfrowego
			digitals = append(digitals, Digital{
				Name:  name,
				Value: bitValue,
			})
		}
	}

	return digitals, nil
}
