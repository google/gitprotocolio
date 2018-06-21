// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitprotocolio

import (
	"fmt"
)

// BytePayloadPacket is the interface of Packets that the payload is []byte.
type BytePayloadPacket interface {
	Packet
	Bytes() []byte
}

// SideBandMainPacket is a sideband packet for the main stream (0x01).
type SideBandMainPacket []byte

// EncodeToPktLine serializes the packet.
func (p SideBandMainPacket) EncodeToPktLine() []byte {
	sz := len(p)
	if sz > 0xFFFF-5 {
		panic("content too large")
	}
	return append([]byte(fmt.Sprintf("%04x%c", sz+5, 1)), p...)
}

// Bytes returns the payload.
func (p SideBandMainPacket) Bytes() []byte {
	return p
}

// SideBandReportPacket is a sideband packet for the report stream (0x02).
type SideBandReportPacket []byte

// EncodeToPktLine serializes the packet.
func (p SideBandReportPacket) EncodeToPktLine() []byte {
	sz := len(p)
	if sz > 0xFFFF-5 {
		panic("content too large")
	}
	return append([]byte(fmt.Sprintf("%04x%c", sz+5, 2)), p...)
}

// Bytes returns the payload.
func (p SideBandReportPacket) Bytes() []byte {
	return p
}

// SideBandErrorPacket is a sideband packet for the error stream (0x03).
type SideBandErrorPacket []byte

// EncodeToPktLine serializes the packet.
func (p SideBandErrorPacket) EncodeToPktLine() []byte {
	sz := len(p)
	if sz > 0xFFFF-5 {
		panic("content too large")
	}
	return append([]byte(fmt.Sprintf("%04x%c", sz+5, 3)), p...)
}

// Bytes returns the payload.
func (p SideBandErrorPacket) Bytes() []byte {
	return p
}

// ParseSideBandPacket parses the BytesPacket as a sideband packet. Returns nil
// if the packet is not a sideband packet.
func ParseSideBandPacket(bp BytesPacket) BytePayloadPacket {
	switch bp[0] {
	case 1:
		return SideBandMainPacket(bp[1:])
	case 2:
		return SideBandReportPacket(bp[1:])
	case 3:
		return SideBandErrorPacket(bp[1:])
	}
	return nil
}
