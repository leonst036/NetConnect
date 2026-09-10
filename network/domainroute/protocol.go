package domainroute

import (
	"encoding/binary"
	"errors"
)

const (
	CmdConnect byte = 0x01
	CmdData    byte = 0x02
	CmdClose   byte = 0x03
)

var (
	ErrFrameTooShort    = errors.New("frame is too short")
	ErrInvalidDomainLen = errors.New("invalid domain length in connect frame")
	ErrUnknownCommand   = errors.New("unknown command")
)

type FrameHeader struct {
	Cmd       byte
	ChannelID uint32
}

func ParseHeader(data []byte) (FrameHeader, error) {
	if len(data) < 5 {
		return FrameHeader{}, ErrFrameTooShort
	}
	return FrameHeader{
		Cmd:       data[0],
		ChannelID: binary.BigEndian.Uint32(data[1:5]),
	}, nil
}

type ConnectFrame struct {
	ChannelID uint32
	Port      uint16
	Domain    string
}

func EncodeConnect(channelID uint32, port uint16, domain string) []byte {
	domainBytes := []byte(domain)
	domainLen := len(domainBytes)
	buf := make([]byte, 9+domainLen)
	buf[0] = CmdConnect
	binary.BigEndian.PutUint32(buf[1:5], channelID)
	binary.BigEndian.PutUint16(buf[5:7], port)
	binary.BigEndian.PutUint16(buf[7:9], uint16(domainLen))
	copy(buf[9:], domainBytes)
	return buf
}

func ParseConnect(data []byte) (ConnectFrame, error) {
	if len(data) < 9 {
		return ConnectFrame{}, ErrFrameTooShort
	}
	if data[0] != CmdConnect {
		return ConnectFrame{}, ErrUnknownCommand
	}
	channelID := binary.BigEndian.Uint32(data[1:5])
	port := binary.BigEndian.Uint16(data[5:7])
	domainLen := int(binary.BigEndian.Uint16(data[7:9]))
	if len(data) < 9+domainLen {
		return ConnectFrame{}, ErrInvalidDomainLen
	}
	domain := string(data[9 : 9+domainLen])
	return ConnectFrame{
		ChannelID: channelID,
		Port:      port,
		Domain:    domain,
	}, nil
}

func EncodeData(channelID uint32, payload []byte) []byte {
	buf := make([]byte, 5+len(payload))
	buf[0] = CmdData
	binary.BigEndian.PutUint32(buf[1:5], channelID)
	copy(buf[5:], payload)
	return buf
}

func ParseData(data []byte) (uint32, []byte, error) {
	if len(data) < 5 {
		return 0, nil, ErrFrameTooShort
	}
	if data[0] != CmdData {
		return 0, nil, ErrUnknownCommand
	}
	channelID := binary.BigEndian.Uint32(data[1:5])
	return channelID, data[5:], nil
}

func EncodeClose(channelID uint32) []byte {
	buf := make([]byte, 5)
	buf[0] = CmdClose
	binary.BigEndian.PutUint32(buf[1:5], channelID)
	return buf
}

func ParseClose(data []byte) (uint32, error) {
	if len(data) < 5 {
		return 0, ErrFrameTooShort
	}
	if data[0] != CmdClose {
		return 0, ErrUnknownCommand
	}
	return binary.BigEndian.Uint32(data[1:5]), nil
}
