package handler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"frame_reductor/model"
)

// ConvertConfigurationFrame modifies the configuration frame and its associated byte data.
func ConvertConfigurationFrame(frame model.C37ConfigurationFrame2, frameData []byte) (model.C37ConfigurationFrame2, []byte, error) {
	// Modify the frame structure
	frame.NumPhasors = 1
	frame.NumAnalogs = 0
	frame.NumDigitals = 0
	frame.ChannelNames = []string{"U_SEQ+"}
	frame.PhasorUnits = []model.PhasorUnit{frame.PhasorUnits[0]}
	frame.AnalogUnits = nil
	frame.DigitalUnits = nil
	frame.DataRate = int16(model.FramesCount)

	// Update the frameData
	var buf bytes.Buffer

	// Write the header
	header := frame.C37Header
	header.FrameSize = calculateNewFrameSize(&frame) //TODO
	if err := binary.Write(&buf, binary.BigEndian, header.Sync); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu Sync: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.FrameSize); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu FrameSize: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.IDCode); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu IDCode: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.Soc); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu Soc: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.FracSec); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu FracSec: %v", err)
	}

	// Write the frame-specific fields
	if err := binary.Write(&buf, binary.BigEndian, frame.TimeBase.TimeMultiplier); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu TimeBase: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, frame.NumPMU); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu NumPMU: %v", err)
	}
	if _, err := buf.Write([]byte(padName(frame.StationName, 16))); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu StationName: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, frame.IDCode2); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu IDCode2: %v", err)
	}
	// Zakoduj pole Format przy użyciu encodeFormatBits
	formatValue := model.EncodeFormatBits(frame.Format)
	if err := binary.Write(&buf, binary.BigEndian, formatValue); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu Format: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, frame.NumPhasors); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu NumPhasors: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, frame.NumAnalogs); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu NumAnalogs: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, frame.NumDigitals); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu NumDigitals: %v", err)
	}

	// Write the channel names
	for _, name := range frame.ChannelNames {
		paddedName := padName(name, 16)
		if _, err := buf.Write([]byte(paddedName)); err != nil {
			return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu ChannelNames: %v", err)
		}
	}
	fmt.Println("Buf after write CHNAM:    ", buf.Bytes())

	// Write the PhasorUnits
	for _, unit := range frame.PhasorUnits {
		// Konwersja ChannelType i ConversionFactor
		channelType := uint8(unit.ChannelType)                  // Konwersja ChannelType na uint8
		conversionFactor := uint32(unit.ConversionFactor * 1e5) // Skalowanie ConversionFactor do 10^-5

		// Zapis ChannelType (1 bajt)
		if err := binary.Write(&buf, binary.BigEndian, channelType); err != nil {
			return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu ChannelType w PhasorUnits: %v", err)
		}

		// Zapis ConversionFactor (3 bajty)
		convFactorBytes := []byte{
			byte((conversionFactor >> 16) & 0xFF), // Najbardziej znaczący bajt
			byte((conversionFactor >> 8) & 0xFF),  // Środkowy bajt
			byte(conversionFactor & 0xFF),         // Najmniej znaczący bajt
		}
		if _, err := buf.Write(convFactorBytes); err != nil {
			return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu ConversionFactor w PhasorUnits: %v", err)
		}
		fmt.Println("Buf after write PHUNIT:   ", buf.Bytes())
	}

	// Write AnalogUnits and DigitalUnits (if present)
	if frame.AnalogUnits != nil {
		for _, unit := range frame.AnalogUnits {
			if err := binary.Write(&buf, binary.BigEndian, unit); err != nil {
				return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu AnalogUnits: %v", err)
			}
		}
	}
	if frame.DigitalUnits != nil {
		for _, unit := range frame.DigitalUnits {
			if err := binary.Write(&buf, binary.BigEndian, unit); err != nil {
				return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu DigitalUnits: %v", err)
			}
		}
	}

	// Write FNom
	encodedFNom := model.EncodeFNom(frame.FNom)
	if err := binary.Write(&buf, binary.BigEndian, encodedFNom); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu FNom: %v", err)
	}

	fmt.Println("Buf after write FNOM:     ", buf.Bytes())
	// Write ConfigCount
	if err := binary.Write(&buf, binary.BigEndian, frame.ConfigCount); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu ConfigCount: %v", err)
	}
	fmt.Println("Buf after write CFGCNT:   ", buf.Bytes())

	// Write DataRate
	if err := binary.Write(&buf, binary.BigEndian, frame.DataRate); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu DataRate: %v", err)
	}
	fmt.Println("Buf after write DATA_RATE:", buf.Bytes())

	// Write CRC
	if err := binary.Write(&buf, binary.BigEndian, frame.CRC); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd zapisu CRC: %v", err)
	}

	// Dodanie 5 bajtów wypełnionych zerami
	padding := make([]byte, 5) // Tworzymy bufor 5 bajtów z zerami
	if _, err := buf.Write(padding); err != nil {
		return model.C37ConfigurationFrame2{}, nil, fmt.Errorf("błąd dodawania 5 bajtów wypełnionych zerami: %v", err)
	}

	// Oblicz nową długość ramki
	frameSize := uint32(buf.Len()) // Długość ramki to długość zapisanych danych

	// Zaktualizuj bajty 3 i 4 w ramce, aby zawierały nową długość
	frameData[2] = byte(frameSize >> 8)   // Bajt 3
	frameData[3] = byte(frameSize & 0xFF) // Bajt 4

	// aktualizacja wartości długości w buforze
	// Resetuj bufor
	var updatedBuf bytes.Buffer
	// Skopiuj dane do nowego bufora
	updatedBuf.Write(buf.Bytes())

	// Zaktualizuj bajty 3 i 4 w nowym buforze
	updatedBuf.Bytes()[2] = byte(frameSize >> 8)   // Bajt 3
	updatedBuf.Bytes()[3] = byte(frameSize & 0xFF) // Bajt 4
	fmt.Println("Buf after update FSIZE: ", updatedBuf.Bytes())

	// Zwróć zaktualizowaną ramkę i dane
	return frame, updatedBuf.Bytes(), nil
}

// ConvertDataFrame modifies the data frame and its associated byte data.
func ConvertDataFrame(frame model.C37DataFrame, frameData []byte) (model.C37DataFrame, []byte, error) {
	// Usuń wszystkie fazory poza 'U_SEQ+'
	var filteredPhasors []model.Phasor
	for _, phasor := range frame.Phasors {
		if phasor.Name == "U_SEQ+" {
			filteredPhasors = append(filteredPhasors, phasor)
		}
	}
	// Zaktualizuj pole Phasors
	frame.Phasors = filteredPhasors

	// Wyczyść Analogs i Digitals
	frame.Analogs = nil
	frame.Digitals = nil

	// Update the frameData
	var buf bytes.Buffer

	// Write the header
	header := frame.C37Header
	//header.FrameSize = calculateNewFrameSize(&frame) //TODO
	if err := binary.Write(&buf, binary.BigEndian, header.Sync); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu Sync: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.FrameSize); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu FrameSize: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.IDCode); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu IDCode: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.Soc); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu Soc: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, header.FracSec); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu FracSec: %v", err)
	}
	// Write the frame-specific fields
	statValue, err := model.EncodeStat(frame.Stat)
	if err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd dekodowania Stat: %v", err)
	}
	if err = binary.Write(&buf, binary.BigEndian, statValue); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu Stat: %v", err)
	}
	fmt.Println("Buf after write STAT:   ", buf.Bytes())

	// Write the Phasors
	phasorData, err := model.EncodePhasors(frame.Phasors)
	fmt.Printf("Kodowane PHASORS: %X %+v\n", phasorData, phasorData)
	if err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd kodowania fazorów: %v", err)
	}
	if _, err := buf.Write(phasorData); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu fazorów: %v", err)
	}
	fmt.Println("Buf after write PHASORS:", buf.Bytes())

	// Write the Frequency
	freqData, err := model.EncodeFrequency(frame.Frequency)
	if err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd kodowania częstotliwości: %v", err)
	}
	if _, err := buf.Write(freqData); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu częstotliwości: %v", err)
	}
	fmt.Println("Buf after write FREQ:   ", buf.Bytes(), "  freq data:  ", freqData)

	// Write the ROCOF
	dfreqData, err := model.EncodeROCOF(frame.Rocof)
	if err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd kodowania ROCOF: %v", err)
	}
	if _, err := buf.Write(dfreqData); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu ROCOF: %v", err)
	}
	fmt.Println("Buf after write DFREQ:  ", buf.Bytes())
	// Write CHK
	if err := binary.Write(&buf, binary.BigEndian, frame.CRC); err != nil {
		return model.C37DataFrame{}, nil, fmt.Errorf("błąd zapisu CHK: %v", err)
	}
	fmt.Println("Buf after write CHK:    ", buf.Bytes())

	// Oblicz nową długość ramki
	frameSize := uint32(buf.Len()) // Długość ramki to długość zapisanych danych

	// Zaktualizuj bajty 3 i 4 w ramce, aby zawierały nową długość
	frameData[2] = byte(frameSize >> 8)   // Bajt 3
	frameData[3] = byte(frameSize & 0xFF) // Bajt 4

	// aktualizacja wartości długości w buforze
	// Resetuj bufor
	var updatedBuf bytes.Buffer
	// Skopiuj dane do nowego bufora
	updatedBuf.Write(buf.Bytes())

	// Zaktualizuj bajty 3 i 4 w nowym buforze
	updatedBuf.Bytes()[2] = byte(frameSize >> 8)   // Bajt 3
	updatedBuf.Bytes()[3] = byte(frameSize & 0xFF) // Bajt 4
	fmt.Println("Buf after update FSIZE: ", updatedBuf.Bytes())

	// Zwróć zaktualizowaną ramkę i dane
	return frame, updatedBuf.Bytes(), nil
}

func calculateNewFrameSize(frame *model.C37ConfigurationFrame2) uint16 {
	size := 18 // Fixed header size
	size += 2  // TimeBase
	size += 2  // NumPMU
	size += 16 // StationName
	size += 2  // IDCode2
	size += 2  // Format
	size += 2  // NumPhasors
	size += 2  // NumAnalogs
	size += 2  // NumDigitals
	size += len(frame.ChannelNames) * 16
	size += len(frame.PhasorUnits) * 4
	size += len(frame.AnalogUnits) * 4
	size += len(frame.DigitalUnits) * 4
	size += 2 // FNom
	size += 2 // DataRate
	size += 2 // ConfigCount
	size += 2 // CRC
	return uint16(size)
}

func padName(name string, length int) string {
	if len(name) > length {
		return name[:length]
	}
	for len(name) < length {
		name += " "
	}
	return name
}
