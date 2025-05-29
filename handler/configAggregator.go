package handler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"frame_reductor/model"
	"log"
	"sync"
)

var (
	configBuffer = make(map[uint16]*model.C37ConfigurationFrame2) // klucz: IDCode PMU
	configMutex  sync.Mutex
	requiredPMUs = 3 // TODO Ilość PMU do zebrania, wartość do ustawienia w zależności od ilości obsługiwanych PMU
)
var analogTypeMap = map[model.AnalogType]uint16{
	"SinglePointOnWave": 0,
	"RMS":               1,
	"Peak":              2,
	"Reserved":          5,  // lub inny wybrany z przedziału 5-64
	"UserDefined":       65, // lub inny wybrany z przedziału 65-255
	"Unknown":           255,
}

func HandleConfigFrame(frame *model.C37ConfigurationFrame2, frameData []byte, frameChan chan []byte) {
	configMutex.Lock()
	defer configMutex.Unlock()

	configBuffer[frame.C37Header.IDCode] = frame
	fmt.Printf("Odebrano ConfigurationFrame2 z PMU ID %d\n", frame.C37Header.IDCode)

	if len(configBuffer) == requiredPMUs {
		fmt.Println("Wszystkie konfiguracje odebrane, buduję agregat...")

		aggFrame, aggFrameBytes, err := BuildAggregatedConfigFrame()
		if err != nil {
			fmt.Println("Błąd budowania agregatu:", err)
			return
		}
		fmt.Printf("Zbudowano ConfigurationFrame2 [%d bytes]: %x\n", len(aggFrameBytes), aggFrameBytes)

		ProcessConfigurationFrame(*aggFrame, frameData, frameChan)
	}
}

func BuildAggregatedConfigFrame() (*model.C37ConfigurationFrame2, []byte, error) {
	var aggFrame model.C37ConfigurationFrame2

	configMutex.Lock()
	defer configMutex.Unlock()

	if len(configBuffer) == 0 {
		return nil, nil, fmt.Errorf("brak ramek do agregacji")
	}

	// Nagłówek z pierwszej ramki (możesz zmienić IDCode na np. 999)
	first := configBuffer[findFirstKey()]
	aggFrame.C37Header = first.C37Header
	aggFrame.C37Header.IDCode = 999 // TODO ID agregatu, można ustawić w parametrze wejściowym

	aggFrame.TimeBase = first.TimeBase
	aggFrame.ConfigCount = first.ConfigCount
	aggFrame.DataRate = first.DataRate
	aggFrame.FNom = first.FNom

	// Zliczanie PMU
	aggFrame.NumPMU = uint16(len(configBuffer))

	// Agregacja PMU
	var pmuFrames []*model.C37ConfigurationFrame2
	for _, cfg := range configBuffer {
		pmuFrames = append(pmuFrames, cfg)

		aggFrame.NumPhasors += cfg.NumPhasors
		aggFrame.NumAnalogs += cfg.NumAnalogs
		aggFrame.NumDigitals += cfg.NumDigitals
	}

	// Serializacja
	var buf bytes.Buffer

	// --- HEADER ---
	binary.Write(&buf, binary.BigEndian, aggFrame.C37Header.Sync)
	binary.Write(&buf, binary.BigEndian, uint16(0)) // FrameSize placeholder
	binary.Write(&buf, binary.BigEndian, aggFrame.C37Header.IDCode)
	binary.Write(&buf, binary.BigEndian, aggFrame.C37Header.Soc)
	binary.Write(&buf, binary.BigEndian, aggFrame.C37Header.FracSec)

	timeBaseRaw := uint32(aggFrame.TimeBase.TimeMultiplier) & 0x7FFF
	binary.Write(&buf, binary.BigEndian, timeBaseRaw)

	binary.Write(&buf, binary.BigEndian, aggFrame.NumPMU)

	// --- CF2 FIELDS: 8-19 ---
	for _, pmu := range pmuFrames {
		// STN (16 bajtów)
		nameBytes := make([]byte, 16)
		copy(nameBytes, []byte(pmu.StationName))
		buf.Write(nameBytes)

		// IDCODE 2
		binary.Write(&buf, binary.BigEndian, pmu.IDCode2)

		// FORMAT
		binary.Write(&buf, binary.BigEndian, pmu.Format.ToUint16())

		// PHNMR, ANNMR, DGNMR
		binary.Write(&buf, binary.BigEndian, pmu.NumPhasors)
		binary.Write(&buf, binary.BigEndian, pmu.NumAnalogs)
		binary.Write(&buf, binary.BigEndian, pmu.NumDigitals)

		// CHNAM
		for _, ch := range pmu.ChannelNames {
			chName := make([]byte, 16)
			copy(chName, []byte(ch))
			buf.Write(chName)
		}

		// PHUNIT (4 bajty na każdy kanał: 2 bajty typ + 2 bajty współczynnik)
		for _, u := range pmu.PhasorUnits {
			binary.Write(&buf, binary.BigEndian, uint16(u.ChannelType)) // 2 bajty typ (np. napięcie/prąd)
			scaled := uint16(u.ConversionFactor * 1e5)                  // IEEE: PhasorUnit w 10^-5 V/A per bit
			binary.Write(&buf, binary.BigEndian, scaled)                // 2 bajty współczynnik
		}

		// ANUNIT (4 bajty na każdy kanał: 2 bajty typ + 2 bajty skala)
		for _, u := range pmu.AnalogUnits {
			value, ok := analogTypeMap[u.ChannelType]
			if !ok {
				log.Printf("Nieznany AnalogType: %s, używam 0", u.ChannelType)
				value = 0
			}
			binary.Write(&buf, binary.BigEndian, value)  // 2 bajty typ (np. napięcie/prąd/inna jednostka)
			scaled := uint16(u.ScalingFactor)            // można skalować jeśli potrzeba
			binary.Write(&buf, binary.BigEndian, scaled) // 2 bajty współczynnik
		}

		// DIGUNIT (4 bajty na każdy blok: 2 bajty normal + 2 bajty off-normal)
		for _, u := range pmu.DigitalUnits {
			binary.Write(&buf, binary.BigEndian, u.NormalStatusMask) // 2 bajty
			binary.Write(&buf, binary.BigEndian, u.ValidInputsMask)  // 2 bajty
		}

		// FNOM
		binary.Write(&buf, binary.BigEndian, pmu.FNom.ToUint16())

		// CFGCNT
		binary.Write(&buf, binary.BigEndian, pmu.ConfigCount)
	}

	// --- DATA_RATE ---
	binary.Write(&buf, binary.BigEndian, aggFrame.DataRate)

	// --- FrameSize ---
	frameSize := uint16(buf.Len() + 2) // +2 bajty na CRC
	binary.BigEndian.PutUint16(buf.Bytes()[2:4], frameSize)

	// --- CRC ---
	crc := model.CalculateCRC(buf.Bytes())
	binary.Write(&buf, binary.BigEndian, crc)

	return &aggFrame, buf.Bytes(), nil
}

func findFirstKey() uint16 {
	for k := range configBuffer {
		return k
	}
	return 0
}
