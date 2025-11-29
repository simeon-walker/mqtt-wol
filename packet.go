package main

////////////////////////////////////////////////////////////////////////////////

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"regexp"
)

////////////////////////////////////////////////////////////////////////////////

var (
	delims = ":-"
	reMAC  = regexp.MustCompile(`^([0-9a-fA-F]{2}[` + delims + `]){5}([0-9a-fA-F]{2})$`)
)

////////////////////////////////////////////////////////////////////////////////

// MACAddress represents a 6 byte network mac address.
type MACAddress [6]byte

// MagicPacket is constituted of 6 bytes of 0xFF followed by 16-groups of the
// destination MAC address.
type MagicPacket struct {
	header  [6]byte
	payload [16]MACAddress
}

// New returns a magic packet based on a mac address string.
func NewMagicPacket(mac string) (*MagicPacket, error) {
	var packet MagicPacket
	var macAddr MACAddress

	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, err
	}

	// We only support 6 byte MAC addresses since it is much harder to use the
	// binary.Write(...) interface when the size of the MagicPacket is dynamic.
	if !reMAC.MatchString(mac) {
		return nil, fmt.Errorf("%s is not a IEEE 802 MAC-48 address", mac)
	}

	// Copy bytes from the returned HardwareAddr -> a fixed size MACAddress.
	for idx := range macAddr {
		macAddr[idx] = hwAddr[idx]
	}

	// Setup the header which is 6 repetitions of 0xFF.
	for idx := range packet.header {
		packet.header[idx] = 0xFF
	}

	// Setup the payload which is 16 repetitions of the MAC addr.
	for idx := range packet.payload {
		packet.payload[idx] = macAddr
	}

	return &packet, nil
}

// This function accepts a MAC address string, and s
// Function to send a magic packet to a given mac address
func (mp *MagicPacket) Send(addr string) error {

	// Fill our byte buffer with the bytes in our MagicPacket
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, mp)

	// Get a UDPAddr to send the broadcast to
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("unable to get UDP address for %s", addr)
	}

	// Open a UDP connection, and defer its cleanup
	connection, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("unable to dial UDP address %s", addr)
	}
	defer connection.Close()

	// Write the bytes of the MagicPacket to the connection
	bytesWritten, err := connection.Write(buf.Bytes())
	if err != nil {
		return errors.New("unable to write packet to connection")
	} else if bytesWritten != 102 {
		slog.Warn("unexpected bytes", "written", bytesWritten, "expected", 102)
	}

	return nil
}
