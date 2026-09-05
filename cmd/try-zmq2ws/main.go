package main

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-zeromq/zmq4"
)

func main() {
	sub := zmq4.NewSub(context.Background(), zmq4.WithDialerRetry(time.Second), zmq4.WithAutomaticReconnect(true), zmq4.WithDialerMaxRetries(-1), zmq4.WithTimeout(time.Second))
	defer sub.Close()

	if err := sub.Dial("tcp://192.168.30.130:5555"); err != nil {
		log.Fatalf("could not dial: %v", err)
	}

	if err := sub.SetOption(zmq4.OptionSubscribe, ""); err != nil {
		log.Fatalf("could not subscribe: %v", err)
	}

	for {
		// Read envelope
		msg, err := sub.Recv()
		if err != nil {
			log.Fatalf("could not receive message: %v", err)
		}
		if len(msg.Frames[1]) != 16 {
			log.Fatalf("metadata length got %v, but want 16", len(msg.Frames[1]))
		}
		nal := msg.Frames[0]
		metadata := msg.Frames[1]
		// endOfFrame := metadata[0]&1 != 0
		timestampNs := binary.LittleEndian.Uint64(metadata[:8])>>1 | uint64(metadata[8]&1)<<63
		_ = timestampNs
		// log.Printf("ts: %v [%v] [%v]: %v, EOF: %v", timestampNs, len(nal), len(metadata), metadata, endOfFrame)
		// log.Printf("[%v]: %v, EOF: %v", len(nal), nal, endOfFrame)
		// log.Printf("ts: %v EOF: %v", timestampNs, endOfFrame)

		nals := parseNal(nal)
		log.Infof("%v", nals)
	}
}

type nalUnitType uint8

const (
	nalUnitTypeTrailN    nalUnitType = 0
	nalUnitTypeTrailR    nalUnitType = 1
	nalUnitTypeTSAN      nalUnitType = 2
	nalUnitTypeTSAR      nalUnitType = 3
	nalUnitTypeSTSAN     nalUnitType = 4
	nalUnitTypeSTSAR     nalUnitType = 5
	nalUnitTypeRADLN     nalUnitType = 6
	nalUnitTypeRADLR     nalUnitType = 7
	nalUnitTypeRASLN     nalUnitType = 8
	nalUnitTypeRASLR     nalUnitType = 9
	nalUnitTypeBLAWLP    nalUnitType = 16
	nalUnitTypeBLAWRADL  nalUnitType = 17
	nalUnitTypeBLANLP    nalUnitType = 18
	nalUnitTypeIDRWRADL  nalUnitType = 19
	nalUnitTypeIDRNLP    nalUnitType = 20
	nalUnitTypeCRA       nalUnitType = 21
	nalUnitTypeVPS       nalUnitType = 32
	nalUnitTypeSPS       nalUnitType = 33
	nalUnitTypePPS       nalUnitType = 34
	nalUnitTypeAUD       nalUnitType = 35
	nalUnitTypeEOS       nalUnitType = 36
	nalUnitTypeEOB       nalUnitType = 37
	nalUnitTypeFD        nalUnitType = 38
	nalUnitTypePrefixSEI nalUnitType = 39
	nalUnitTypeSuffixSEI nalUnitType = 40
)

type nalUnit struct {
	forbiddenZeroBit   uint8
	nalUnitType        nalUnitType
	nuhLayerID         uint8
	nuhTemporalIDPlus1 uint8
	data               []byte
}

func parseNal(rawData []byte) []nalUnit {
	if len(rawData) < 2 {
		return nil
	}

	nals := make([]nalUnit, 0, 4)
	payloadStart := -1
	zeroRun := 0

	for i, b := range rawData {
		if b == 0 {
			zeroRun++
			continue
		}
		if b == 1 && zeroRun >= 2 {
			startCode := i - zeroRun
			if payloadStart >= 0 && startCode-payloadStart >= 2 {
				nals = append(nals, parseNalUnit(rawData[payloadStart:startCode]))
			}
			payloadStart = i + 1
		}
		zeroRun = 0
	}

	if payloadStart < 0 {
		nals = append(nals, parseNalUnit(rawData))
	} else if len(rawData)-payloadStart >= 2 {
		nals = append(nals, parseNalUnit(rawData[payloadStart:]))
	}

	return nals
}

func parseNalUnit(rawData []byte) nalUnit {
	return nalUnit{
		forbiddenZeroBit:   rawData[0] >> 7,
		nalUnitType:        nalUnitType(rawData[0] >> 1 & 0x3f),
		nuhLayerID:         (rawData[0]&1)<<5 | rawData[1]>>3,
		nuhTemporalIDPlus1: rawData[1] & 7,
		data:               rawData[2:],
	}
}
