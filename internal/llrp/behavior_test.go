//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"
	"testing/quick"
)

func TestImpinjEnableBool16(t *testing.T) {
	if err := quick.Check(func(subtype uint32) bool {
		data := impinjEnableBool16(subtype)
		if len(data) != int(binary.BigEndian.Uint16(data[2:])) {
			return false
		}

		if binary.BigEndian.Uint16(data) != 1023 {
			return false
		}

		if binary.BigEndian.Uint32(data[4:]) != uint32(PENImpinj) {
			return false
		}

		if binary.BigEndian.Uint32(data[8:]) != subtype {
			return false
		}

		if binary.BigEndian.Uint16(data[12:]) != 1 {
			return false
		}

		return true
	}, nil); err != nil {
		t.Error(err)
	}
}

var fccFreqs = []Kilohertz{
	902750, 903250, 903750, 904250, 904750, 905250, 905750,
	906250, 906750, 907250, 907750, 908250, 908750, 909250,
	909750, 910250, 910750, 911250, 911750, 912250, 912750,
	913250, 913750, 914250, 914750, 915250, 915750, 916250,
	916750, 917250, 917750, 918250, 918750, 919250, 919750,
	920250, 920750, 921250, 921750, 922250, 922750, 923250,
	923750, 924250, 924750, 925250, 925750, 926250, 926750,
	927250,
}

func newImpinjCaps() *GetReaderCapabilitiesResponse {
	powerTable := make([]TransmitPowerLevelTableEntry, 81)
	for i := range powerTable {
		powerTable[i] = TransmitPowerLevelTableEntry{
			Index:              uint16(i + 1),
			TransmitPowerValue: MillibelMilliwatt(i*25 + 1000),
		}
	}

	receiveTable := make([]ReceiveSensitivityTableEntry, 42)
	for i := range receiveTable {
		receiveTable[i] = ReceiveSensitivityTableEntry{
			Index:              uint16(i + 1),
			ReceiveSensitivity: uint16(i + 9),
		}
	}
	receiveTable[0].ReceiveSensitivity = 0

	if len(fccFreqs) != 50 {
		panic("This code is based on the assumption that the FCC Frequency list has 50 entries.")
	}

	// Build the frequency list by appending entries from the FCC list,
	// where we step the index into the FCC list by some generator of Z/Z(50)
	// I've chosen 7, but other good options are 3, 9, 11, 13, 17, 21, etc.
	// The important thing is just that it doesn't have a factor of 2 or 5.
	hopTableFreqs := make([]Kilohertz, 50)
	for i, j := 0, 0; i < 50; i, j = i+1, (j+7)%50 {
		hopTableFreqs[i] = fccFreqs[j]
	}

	const numAntennas = 4
	airProto := make([]PerAntennaAirProtocol, numAntennas)
	for i := range airProto {
		airProto[i] = PerAntennaAirProtocol{
			AntennaID:      AntennaID(i + 1),
			AirProtocolIDs: []AirProtocolIDType{AirProtoEPCGlobalClass1Gen2},
		}
	}

	return &GetReaderCapabilitiesResponse{
		GeneralDeviceCapabilities: &GeneralDeviceCapabilities{
			HasUTCClock:            true,
			DeviceManufacturer:     uint32(PENImpinj),
			Model:                  uint32(SpeedwayR420),
			FirmwareVersion:        "5.14.0.240",
			GPIOCapabilities:       GPIOCapabilities{4, 4},
			MaxSupportedAntennas:   numAntennas,
			PerAntennaAirProtocols: airProto,
			ReceiveSensitivities:   receiveTable,
		},

		LLRPCapabilities: &LLRPCapabilities{
			CanReportBufferFillWarning:          true,
			SupportsEventsAndReportHolding:      true,
			MaxPriorityLevelSupported:           1,
			MaxROSpecs:                          1,
			MaxSpecsPerROSpec:                   32,
			MaxInventoryParameterSpecsPerAISpec: 1,
			MaxAccessSpecs:                      1508,
			MaxOpSpecsPerAccessSpec:             8,
		},

		C1G2LLRPCapabilities: &C1G2LLRPCapabilities{
			SupportsBlockWrite:       true,
			MaxSelectFiltersPerQuery: 2,
		},

		RegulatoryCapabilities: &RegulatoryCapabilities{
			CountryCode:            840,
			CommunicationsStandard: 1,
			UHFBandCapabilities: &UHFBandCapabilities{
				TransmitPowerLevels: powerTable,
				FrequencyInformation: FrequencyInformation{
					Hopping: true,
					FrequencyHopTables: []FrequencyHopTable{{
						HopTableID:  1,
						Frequencies: hopTableFreqs,
					}},
				},

				C1G2RFModes: UHFC1G2RFModeTable{
					UHFC1G2RFModeTableEntries: []UHFC1G2RFModeTableEntry{
						{
							ModeID:                0,
							DivideRatio:           DRSixtyFourToThree,
							Modulation:            FM0,
							ForwardLinkModulation: PhaseReversalASK,
							SpectralMask:          SpectralMaskMultiInterrogator,
							BackscatterDataRate:   640000, // actually BLF
							PIERatio:              1500,
							MinTariTime:           6250,
							MaxTariTime:           6250,
						},

						{
							ModeID:                1,
							DivideRatio:           DRSixtyFourToThree,
							Modulation:            Miller2,
							ForwardLinkModulation: PhaseReversalASK,
							SpectralMask:          SpectralMaskMultiInterrogator,
							BackscatterDataRate:   640000, // actually BLF
							PIERatio:              1500,
							MinTariTime:           6250,
							MaxTariTime:           6250,
						},

						{
							ModeID:                2,
							DivideRatio:           DRSixtyFourToThree,
							Modulation:            Miller4,
							ForwardLinkModulation: DoubleSidebandASK,
							SpectralMask:          SpectralMaskDenseInterrogator,
							BackscatterDataRate:   274000, // actually BLF
							PIERatio:              2000,
							MinTariTime:           20000,
							MaxTariTime:           20000,
						},

						{
							ModeID:                3,
							DivideRatio:           DRSixtyFourToThree,
							Modulation:            Miller8,
							ForwardLinkModulation: DoubleSidebandASK,
							SpectralMask:          SpectralMaskDenseInterrogator,
							BackscatterDataRate:   170600, // actually BLF
							PIERatio:              2000,
							MinTariTime:           20000,
							MaxTariTime:           20000,
						},

						{
							ModeID:                4,
							DivideRatio:           DRSixtyFourToThree,
							Modulation:            Miller4,
							ForwardLinkModulation: DoubleSidebandASK,
							SpectralMask:          SpectralMaskMultiInterrogator,
							BackscatterDataRate:   640000, // actually BLF
							PIERatio:              1500,
							MinTariTime:           7140,
							MaxTariTime:           7140,
						},

						// the rest of these are the "auto modes",
						// so the details are non-sense,
						// but they do store the min values in the fields.
						{
							ModeID:              1000,
							BackscatterDataRate: 40000, PIERatio: 1500,
							MinTariTime: 6250, MaxTariTime: 6250,
						},

						{
							ModeID:              1002,
							BackscatterDataRate: 40000, PIERatio: 1500,
							MinTariTime: 6250, MaxTariTime: 6250,
						},

						{
							ModeID:              1003,
							BackscatterDataRate: 40000, PIERatio: 1500,
							MinTariTime: 6250, MaxTariTime: 6250,
						},

						{
							ModeID:              1004,
							BackscatterDataRate: 40000, PIERatio: 1500,
							MinTariTime: 6250, MaxTariTime: 6250,
						},

						{
							ModeID:              1005,
							BackscatterDataRate: 40000, PIERatio: 1500,
							MinTariTime: 6250, MaxTariTime: 6250,
						},
					},
				},
			},
		},
	}
}

func TestImpinjDevice_invalid(t *testing.T) {
	caps := newImpinjCaps()
	caps.RegulatoryCapabilities.UHFBandCapabilities.C1G2RFModes.UHFC1G2RFModeTableEntries = nil
	_, err := NewImpinjDevice(caps)
	if err == nil {
		t.Fatal("there should be an error if the capabilities lacks an RF Mode Table")
	}
}

func TestImpinjDevice_NewConfig(t *testing.T) {
	caps := newImpinjCaps()
	d, err := NewImpinjDevice(caps)

	if err != nil {
		t.Fatal(err)
	}

	if d.stateAware {
		t.Errorf("d should not be state aware")
	}

	if !d.allowsHop {
		t.Errorf("d should allow hopping")
	}

	if d.nSpecsPerRO != caps.LLRPCapabilities.MaxSpecsPerROSpec {
		t.Errorf("mismatched nRoSpecs: %d != %d",
			d.nSpecsPerRO, caps.LLRPCapabilities.MaxSpecsPerROSpec)
	}

	nFreqs := len(caps.RegulatoryCapabilities.UHFBandCapabilities.
		FrequencyInformation.FrequencyHopTables[0].Frequencies)
	if int(d.nFreqs) != nFreqs {
		t.Errorf("mismatched nFreqs: %d != %d", d.nFreqs, nFreqs)
	}

	if d.nGPIs != caps.GeneralDeviceCapabilities.GPIOCapabilities.NumGPIs {
		t.Errorf("mismatched nGPIs: %d != %d",
			d.nGPIs, caps.GeneralDeviceCapabilities.GPIOCapabilities.NumGPIs)
	}

	if len(d.modes) != 5 {
		t.Errorf("expected 5 modes; got %d", len(d.modes))
	}

	for i := range d.pwrMinToMax[1:] {
		if d.pwrMinToMax[i].TransmitPowerValue > d.pwrMinToMax[i+1].TransmitPowerValue {
			t.Errorf("power levels are not sorted properly: %v", d.pwrMinToMax)
		}

		// The next few tests assume power values are more than 0.01 dBm apart.
		// They
		pwrAtI := d.pwrMinToMax[i]

		// If we look for a power entry that exists, we should get that exact value.
		pIdx, pValue := d.findPower(pwrAtI.TransmitPowerValue)
		if pIdx == 0 {
			t.Errorf("a valid power index should never be 0")
		}
		if pwrAtI.TransmitPowerValue != pValue || pwrAtI.Index != pIdx {
			t.Errorf("expected power %d (index %d), but got %d (index %d)",
				pwrAtI.TransmitPowerValue, pwrAtI.Index, pValue, pIdx)
		}

		// If we search for a power just above this one,
		// we should get back the same power value,
		// assuming power values are more than 0.01 dBm apart.
		pIdx, pValue = d.findPower(pwrAtI.TransmitPowerValue + 1)
		if pwrAtI.Index != pIdx {
			t.Errorf("expected power %d (index %d), but got %d (index %d)",
				pwrAtI.TransmitPowerValue, pwrAtI.Index, pValue, pIdx)
		}
		pIdx, pValue = d.findPower(pwrAtI.TransmitPowerValue + 1)
		if pIdx == 0 {
			t.Errorf("a valid power index should never be 0")
		}

		// If we search for a power just below this one,
		// we should get back a power value less the target,
		if pwrAtI.TransmitPowerValue+1 < pValue {
			t.Errorf("selected power should not exceed %d (index %d), but got %d (index %d)",
				pwrAtI.TransmitPowerValue, pwrAtI.Index, pValue, pIdx)
		}
	}
}

func TestMarshalBehaviorText(t *testing.T) {
	// These tests are really just a sanity check
	// to validate assumptions about json marshaling.
	// They just marshal the interface v to JSON
	// and verify the data matches,
	// then unmarshal that back to a new pointer
	// with the same type as v,
	// and validates it matches the original value.

	tests := []struct {
		name       string
		val        interface{}
		data       []byte
		shouldFail bool
	}{
		{"fast", ScanFast, []byte(`"Fast"`), false},
		{"normal", ScanNormal, []byte(`"Normal"`), false},
		{"deep", ScanDeep, []byte(`"Deep"`), false},
		{"unknownScan", ScanType(501), nil, true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(testCase.val)
			if testCase.shouldFail {
				if err == nil {
					t.Errorf("expected a marshaling error, but got %v", got)
				}
				return
			}

			if !bytes.Equal(got, testCase.data) {
				t.Errorf("got = %s, want %s", got, testCase.data)
			}

			newInst := reflect.New(reflect.TypeOf(testCase.val))
			ptr := newInst.Interface()
			if err := json.Unmarshal(testCase.data, ptr); err != nil {
				t.Errorf("unmarshaling failed: data = %s, error = %v", testCase.data, err)
				return
			}

			newVal := newInst.Elem().Interface()
			if !reflect.DeepEqual(newVal, testCase.val) {
				t.Errorf("roundtrip failed: got = %+v, want %+v", newVal, testCase.val)
			}
		})
	}
}
