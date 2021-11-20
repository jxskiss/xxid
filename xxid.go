package xxid

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
	"unsafe"
)

// ID is a specific kind of unique identifier.
//
// The marshaled form of ID values are ordered by their generation time
// (in millisecond precision).
// Additionally, you may encode some machine information or even some
// business information into an ID.
//
// An ID consists of the following parts:
// 1. millisecond timestamp;
// 2. machine ID, which may be the host identifier, an IP address or user
// specified bytes;
// 3. process ID or user specified port number;
// 4. a counter, starts at a random value;
// 5. a flag value, random or user specified;
//
// An ID can be encoded into binary, base62 or string form. The binary form
// is short and takes less space, the base62 form encodes the binary form
// with base62 encoding, and the string form is friendly to human, it
// encodes the ID using number digits and hex, user can read the content
// of an ID from the string representation, the string form is good for
// scenes where user may need to inspect the content frequently (e.g.
// logging or tracing identifiers).
type ID struct {
	timeMsec  int64
	pidOrPort uint16
	counter   uint16
	flag      uint16
	mIDType   MachineIDType
	machineID [16]byte
}

// MachineIDType indicates the type of and ID's machine ID.
type MachineIDType uint8

const (
	// Random indicates the machine ID is auto-generated random bytes,
	// it's used only when the host identifier can not be read from the
	// operating system.
	Random MachineIDType = 0

	// HostID indicates the machine ID is the hash digest of the host
	// identifier read from the operating system.
	HostID MachineIDType = 1

	// IPv4 indicates the machine ID is an IPv4 address specified by user.
	IPv4 MachineIDType = 2

	// IPv6 indicates the machine ID is an IPv6 address specified by user.
	IPv6 MachineIDType = 3

	// Specified4 indicates the machine ID is a 4 bytes value specified
	// by user.
	Specified4 MachineIDType = 4

	// Specified8 indicates the machine ID is an 8 bytes value specified
	// by user.
	Specified8 MachineIDType = 5

	// Specified16 indicates the machine ID is a 16 bytes value specified
	// by user.
	Specified16 MachineIDType = 6
)

const maxMachineIDType = Specified16

const (
	minBinEncodedLen    = 16
	maxBinEncodedLen    = 28
	minBase62EncodedLen = 22
	maxBase62EncodedLen = 38
	minStringEncodedLen = 38
)

const flagMask = 1 << 15

var (
	machineIdLength  = [...]int{4, 4, 4, 16, 4, 8, 16}
	binEncodedLength = [...]int{16, 16, 16, 28, 16, 20, 28}
	b62EncodedLength = [...]int{22, 22, 22, 38, 22, 27, 38}
	strEncodedLength = [...]int{38, 38, 38, 62, 38, 46, 62}
	binDecodedLength = [...]int{22: 16, 27: 20, 38: 28}
)

var zeroID ID

var (
	errIncorrectBinaryLength = errors.New("xxid: length of binary form is incorrect")
	errIncorrectBase62Length = errors.New("xxid: length of base62 form is incorrect")
	errIncorrectStringLength = errors.New("xxid: length of string form is incorrect")
	errInvalidStringRepr     = errors.New("xxid: string representation is invalid")
	errInvalidJSONString     = errors.New("xxid: JSON string is invalid")
	errUnknownMachineIDType  = errors.New("xxid: machine ID type is unknown")
)

func errInvalidBase62Character(char byte) error {
	return fmt.Errorf("xxid: base62 character %v is invalid", char)
}

var errUnsupportedMachineIDLength = errors.New("xxid: length of specified machine ID is unsupported")

var beEnc = binary.BigEndian

// New generates a unique ID.
func New() ID {
	timeMsec, incr := readTimeAndCounter()
	return newID(defaultGenerator, timeMsec, incr)
}

// NewWithTime generates an ID with the given time.
func NewWithTime(t time.Time) ID {
	timeMsec := t.UnixNano() / 1e6
	incr := incrCounter()
	return newID(defaultGenerator, timeMsec, incr)
}

func newID(gen *Generator, timeMsec int64, counter uint16) ID {
	var id = ID{
		timeMsec:  timeMsec,
		pidOrPort: gen.pidOrPort,
		counter:   counter,
		flag:      gen.flag,
		mIDType:   gen.mIDType,
		machineID: gen.machineID,
	}
	if id.flag == 0 {
		id.flag = randFlag()
	}
	return id
}

// SetFlag returns a new ID value with the given flag.
//
// Note that the function receiver is an ID value, which means that the
// receiver ID value won't be changed, the caller need to save the
// returned value to somewhere, calling this function without using the
// returned value is a no-op.
func (id ID) SetFlag(flag uint16) ID {
	id.flag = flag | flagMask
	return id
}

// Flag returns the ID's flag value.
func (id ID) Flag() uint16 {
	if id.flag&flagMask == 0 {
		return 0
	}
	return id.flag & ^uint16(flagMask)
}

// Time returns the ID's time value.
func (id ID) Time() time.Time {
	return time.Unix(0, id.timeMsec*1e6)
}

// MachineIDType returns the ID's machine ID type.
func (id ID) MachineIDType() MachineIDType {
	return id.mIDType
}

// MachineID returns the ID's machine ID in bytes. The returned bytes may
// be of length 4, 8, or 16 according to the machine ID type.
func (id ID) MachineID() []byte {
	return id.machineID[:machineIdLength[id.mIDType]]
}

// IP returns the ID's machine ID as an IP, the return value may be
// an IPv4 address or IPv6 address.
//
// If the machine ID is not an IP address, it returns nil.
func (id ID) IP() net.IP {
	switch id.mIDType {
	case IPv4:
		var ip [net.IPv4len]byte
		copy(ip[:], id.machineID[:net.IPv4len])
		return ip[:]
	case IPv6:
		var ip [net.IPv6len]byte
		copy(ip[:], id.machineID[:net.IPv6len])
		return ip[:]
	}
	return nil
}

// Pid returns the ID's pid value, note that the returned value may
// be a port number if the Generator is configured by UsePort.
func (id ID) Pid() uint16 {
	return id.pidOrPort
}

// Port returns the ID's port number, note that the returned value may
// be a pid if the Generator is not configured by UsePort.
func (id ID) Port() uint16 {
	return id.pidOrPort
}

// IPPortAddr returns and address string consists of the IP address and
// the port number if the machine ID is an IP address, else it returns "".
func (id ID) IPPortAddr() string {
	switch id.mIDType {
	case IPv4:
		port := strconv.FormatInt(int64(id.Port()), 10)
		ip := net.IP(id.machineID[:4]).String()
		return ip + ":" + port
	case IPv6:
		port := strconv.FormatInt(int64(id.Port()), 10)
		ip := net.IP(id.machineID[:16]).String()
		return "[" + ip + "]:" + port
	}
	return ""
}

// Counter returns the ID's counter value.
func (id ID) Counter() uint16 {
	return id.counter
}

// Short returns the time and counter value of the ID as an int64, the
// returned value is guaranteed to be unique inside a process, even the
// clock has been turned back or leap second happens.
func (id ID) Short() int64 {
	return id.timeMsec<<16 | int64(id.counter)
}

func (id ID) encodeBinary() []byte {
	out := make([]byte, binEncodedLength[id.mIDType])
	offset := 0

	// timestamp since epoch and machine ID type, 6 bytes
	beEnc.PutUint64(out[:8], (uint64(id.timeMsec)<<3)|uint64(id.mIDType))
	copy(out[:6], out[2:8])
	offset += 6
	// increment, 2 bytes
	beEnc.PutUint16(out[offset:offset+2], id.counter)
	offset += 2
	// machine ID
	switch id.mIDType {
	case Random, HostID, IPv4, Specified4:
		copy(out[offset:offset+4], id.machineID[:4])
		offset += 4
	case Specified8:
		copy(out[offset:offset+8], id.machineID[:8])
		offset += 8
	case IPv6, Specified16:
		copy(out[offset:offset+16], id.machineID[:16])
		offset += 16
	}
	// pid or port number, 2 bytes
	beEnc.PutUint16(out[offset:offset+2], id.pidOrPort)
	offset += 2
	// flag, 2 bytes
	beEnc.PutUint16(out[offset:offset+2], id.flag)
	return out
}

func decodeBinary(src []byte) (ID, error) {
	var id ID
	inputLen := len(src)
	if inputLen < minBinEncodedLen {
		return zeroID, errIncorrectBinaryLength
	}

	// timestamp and machine ID type, 6 bytes
	tmp := beEnc.Uint64(src[:8]) >> 16
	id.timeMsec = int64(tmp >> 3)
	id.mIDType = MachineIDType(tmp & 7)
	if id.mIDType > maxMachineIDType {
		return zeroID, errUnknownMachineIDType
	}
	if inputLen != binEncodedLength[id.mIDType] {
		return zeroID, errIncorrectBinaryLength
	}

	// increment, 2 bytes
	id.counter = beEnc.Uint16(src[6:8])

	offset := 8

	// machine ID
	mIdLen := machineIdLength[id.mIDType]
	copy(id.machineID[:], src[offset:offset+mIdLen])
	offset += mIdLen

	// pid or port number, 2 bytes
	id.pidOrPort = beEnc.Uint16(src[offset : offset+2])
	offset += 2

	// flag, 2 bytes
	id.flag = beEnc.Uint16(src[offset : offset+2])

	return id, nil
}

// Binary encodes the ID into its binary form. The returned bytes may
// be of length 16, 20, or 28 according to the machine ID type.
func (id ID) Binary() []byte {
	return id.encodeBinary()
}

// Base62 encodes the ID into its base62 form. The returned bytes may
// be of length 22, 27, or 38 according to the machine ID type.
func (id ID) Base62() []byte {
	buf := id.encodeBinary()
	out := make([]byte, b62EncodedLength[id.mIDType])
	encodeBase62(out, buf)
	return out
}

// String encodes the ID into its string form. The returned string may
// be of length 38, 46, or 62 according to the machine ID type,
func (id ID) String() string {
	var out = make([]byte, strEncodedLength[id.mIDType])
	var tmp [2]byte

	// timestamp
	t := time.Unix(0, id.timeMsec*1e6)
	msec := id.timeMsec % 1000
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	int2byte(out[:4], year)
	int2byte(out[4:6], int(month))
	int2byte(out[6:8], day)
	int2byte(out[8:10], hour)
	int2byte(out[10:12], minute)
	int2byte(out[12:14], second)
	int2byte(out[14:17], int(msec))

	// flag
	beEnc.PutUint16(tmp[:2], id.flag)
	hex.Encode(out[17:21], tmp[:2])

	// machine ID type
	int2byte(out[21:22], int(id.mIDType))

	offset := 22

	// machine ID
	mIdLen := machineIdLength[id.mIDType]
	hex.Encode(out[offset:offset+mIdLen*2], id.machineID[:mIdLen])
	offset += mIdLen * 2

	// pid or port number
	// increment
	for _, x := range []uint16{
		id.pidOrPort,
		id.counter,
	} {
		beEnc.PutUint16(tmp[:2], x)
		hex.Encode(out[offset:offset+4], tmp[:2])
		offset += 4
	}

	return b2s(out)
}

// MarshalJSON encodes ID to a JSON string using its base62 form.
func (id ID) MarshalJSON() ([]byte, error) {
	buf := id.encodeBinary()
	out := make([]byte, b62EncodedLength[id.mIDType]+2)
	encodeBase62(out[1:len(out)-1], buf[:])
	out[0], out[len(out)-1] = '"', '"'
	return out, nil
}

// UnmarshalJSON decodes ID from a JSON string in its base62 form.
func (id *ID) UnmarshalJSON(buf []byte) error {
	if len(buf) < 2 || buf[0] != '"' || buf[len(buf)-1] != '"' {
		return errInvalidJSONString
	}
	tmp, err := ParseBase62(buf[1 : len(buf)-1])
	if err != nil {
		return err
	}
	*id = tmp
	return nil
}

// ParseBinary parses an ID from its binary form.
func ParseBinary(src []byte) (ID, error) {
	return decodeBinary(src)
}

// ParseBase62 parses an ID from its base62 form.
func ParseBase62(src []byte) (ID, error) {
	inputLen := len(src)
	if inputLen < minBase62EncodedLen || inputLen > maxBase62EncodedLen {
		return zeroID, errIncorrectBase62Length
	}
	binLen := binDecodedLength[inputLen]
	if binLen == 0 {
		return zeroID, errIncorrectBase62Length
	}

	buf := make([]byte, binLen)
	err := decodeBase62(buf, src)
	if err != nil {
		return zeroID, err
	}
	return decodeBinary(buf)
}

// ParseString parses an ID from its string form.
func ParseString(str string) (ID, error) {
	var id ID
	inputLen := len(str)
	if inputLen < minStringEncodedLen {
		return zeroID, errIncorrectStringLength
	}
	machineIdType := MachineIDType(str[21] - '0')
	if machineIdType < 0 || machineIdType > maxMachineIDType {
		return zeroID, errUnknownMachineIDType
	}
	if inputLen != strEncodedLength[machineIdType] {
		return zeroID, errIncorrectStringLength
	}

	var err error
	var tmp [2]byte

	var parseTimestamp = func(buf string) (timeMsec int64, err error) {
		layout := "20060102150405"
		t, err := time.ParseInLocation(layout, buf[:14], time.Local)
		if err != nil {
			return
		}
		msec, err := strconv.ParseInt(buf[14:17], 10, 0)
		if err != nil {
			return
		}
		return t.Unix()*1e3 + msec, nil
	}
	var parseUint16 = func(buf string) (uint16, error) {
		_, err = hex.Decode(tmp[:2], s2b(buf))
		if err != nil {
			return 0, err
		}
		return beEnc.Uint16(tmp[:2]), nil
	}

	// timestamp, 17 bytes
	id.timeMsec, err = parseTimestamp(str[:17])
	if err != nil {
		return zeroID, errInvalidStringRepr
	}

	// flag 2, bytes
	id.flag, err = parseUint16(str[17:21])
	if err != nil {
		return zeroID, errInvalidStringRepr
	}

	// machine ID type, 1 byte
	id.mIDType = machineIdType

	offset := 22

	// machine ID
	mIdLen := machineIdLength[machineIdType]
	_, err = hex.Decode(id.machineID[:mIdLen], s2b(str[offset:offset+mIdLen*2]))
	if err != nil {
		return zeroID, errInvalidStringRepr
	}
	offset += mIdLen * 2

	// pid or port number, 4 bytes
	// increment, 4 bytes
	for _, x := range []*uint16{
		&id.pidOrPort,
		&id.counter,
	} {
		_, err = hex.Decode(tmp[:2], s2b(str[offset:offset+4]))
		if err != nil {
			return zeroID, errInvalidStringRepr
		}
		*x = beEnc.Uint16(tmp[:2])
		offset += 4
	}

	return id, nil
}

func int2byte(bs []byte, val int) {
	size := 10
	l := len(bs) - 1
	for idx := l; idx >= 0; idx-- {
		bs[idx] = byte(uint(val%size) + uint('0'))
		val = val / size
	}
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) []byte {
	type sliceHeader struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	}
	type stringHeader struct {
		Data unsafe.Pointer
		Len  int
	}

	sh := (*stringHeader)(unsafe.Pointer(&s))
	bh := &sliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(bh))
}

//go:noescape
//go:linkname runtime_fastrand runtime.fastrand
func runtime_fastrand() uint32
