package xxid

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"
)

// Code modified from the original project github.com/rs/xid

// ID represents a unique request id, consists of
// - 4 bytes seconds from epoch
// - 1 byte flag / idc
// - 4 bytes machine id
// - 2 bytes pid
// - 4 bytes counter (low 31 bits)
type ID [rawLen]byte

const (
	rawLen     = 15 // binary raw len
	encodedLen = 20 // string encoded len
	uuidLen    = 36 // hyphen connected uuid len

	// epoch starts more recently so that the 32-bit number space gives a
	// significantly higher useful lifetime of around 136 years from July 2017.
	// This number (15e8) was picked to be easy to remember.
	epoch = 1500000000 // 2017-07-14T02:40:00Z
)

var (
	// ErrInvalidID is returned when trying to unmarshal an invalid id.
	ErrInvalidID = errors.New("xxid: invalid ID")

	// defaultGenerator is used for the global New and NewWithTime functions.
	defaultGenerator *Generator
	nilID            ID
)

func init() {
	defaultGenerator = &Generator{
		machineID: readMachineID(),
		pid:       uint16(os.Getpid()),
		flag:      randFlag(),
		counter:   randCounter(),
	}
	// If /proc/self/cpuset exists and is not /, we can assume that we are in a
	// form of container and use the content of cpuset xor-ed with the PID in
	// order get a reasonable machine global unique PID.
	b, err := ioutil.ReadFile("/proc/self/cpuset")
	if err == nil && len(b) > 1 {
		defaultGenerator.pid ^= uint16(crc32.ChecksumIEEE(b))
	}
}

func randFlag() uint8 {
	b := make([]byte, 1)
	if _, err := rand.Reader.Read(b); err != nil {
		panic(fmt.Errorf("xxid: cannot generate random number: %v", err))
	}
	// Always mark random flag's highest bit to distinct from user specified flag.
	return b[0] | 0x80
}

// randCounter generates a random uint32 to be used as initial counter value.
func randCounter() uint32 {
	b := make([]byte, 4)
	if _, err := rand.Reader.Read(b); err != nil {
		panic(fmt.Errorf("xxid: cannot generate random number: %v", err))
	}
	return uint32(b[0]&0x3f)<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// readMachineID generates machine id and puts it into the machineId global
// variable. If this function fails to get the hostname, it will cause a
// runtime error.
func readMachineID() [4]byte {
	var id [4]byte
	hid, err := readPlatformMachineID()
	if err != nil || len(hid) == 0 {
		hid, err = os.Hostname()
	}
	if err == nil && len(hid) != 0 {
		hw := md5.New()
		hw.Write([]byte(hid))
		copy(id[:], hw.Sum(nil))
	} else {
		// Fallback to rand number if machine id can't be gathered.
		if _, err = rand.Read(id[:]); err != nil {
			panic(fmt.Errorf("xxid: cannot get hostname nor generate a random number: %v; %v", err, err))
		}
	}
	return id
}

// New generates a globally unique id.
func New() ID {
	return newID(defaultGenerator, time.Now())
}

// NewWithTime generates a globally unique id with the given time.
func NewWithTime(t time.Time) ID {
	return newID(defaultGenerator, t)
}

func newID(g *Generator, t time.Time) ID {
	var id ID
	// timestamp since epoch, 4 bytes, big endian
	binary.BigEndian.PutUint32(id[:], uint32(t.Unix()-epoch))
	// 1 byte flag
	id[4] = g.flag
	// machine id, 4 bytes, big endian
	copy(id[5:], g.machineID[:])
	// pid, 2 bytes, big endian
	binary.BigEndian.PutUint16(id[9:11], g.pid)
	// increment, 4 bytes (low 31 bits), big endian
	incr := atomic.AddUint32(&g.counter, 1) & 0x7fffffff
	binary.BigEndian.PutUint32(id[11:15], incr)
	return id
}

// String returns a base62 hex lowercased with no padding representation of the id.
func (id ID) String() string {
	text := make([]byte, encodedLen)
	encodeBase62(text, id[:])
	// no need to copy the memory, it's safe
	return *(*string)(unsafe.Pointer(&text))
}

func (id ID) UUID() string {
	text := make([]byte, uuidLen)
	hex.Encode(text[6:], id[:])
	copy(text[0:8], text[6:14])
	text[8] = '-'
	copy(text[9:13], text[14:18])
	text[13] = '-'
	copy(text[14:18], text[18:22])
	text[18] = '-'
	copy(text[19:23], text[22:26])
	text[23] = '-'
	text[24], text[25] = '0', '0'
	// no need to copy text[26:36]
	return *(*string)(unsafe.Pointer(&text))
}

// MarshalText implements encoding/text TextMarshaler interface.
func (id ID) MarshalText() ([]byte, error) {
	text := make([]byte, encodedLen)
	encodeBase62(text, id[:])
	return text, nil
}

// MarshalJSON implements encoding/json Marshaler interface.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsNil() {
		return []byte("null"), nil
	}
	text, err := id.MarshalText()
	return []byte(`"` + string(text) + `"`), err
}

// UnmarshalText implements encoding/text TextUnmarshaler interface.
func (id *ID) UnmarshalText(text []byte) error {
	if len(text) != encodedLen {
		return ErrInvalidID
	}
	if bytes.Compare(text, maxEncoded) > 0 {
		return ErrInvalidID
	}
	decodeBase62(id[:], text)
	return nil
}

// UnmarshalJSON implements encoding/json Unmarshaler interface.
func (id *ID) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		*id = nilID
		return nil
	}
	return id.UnmarshalText(b[1 : len(b)-1])
}

// Time returns the timestamp part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) Time() time.Time {
	// First 4 bytes of ID is 32-bit big-endian seconds from epoch.
	secs := int64(binary.BigEndian.Uint32(id[0:4]))
	return time.Unix(secs+epoch, 0)
}

// Flag returns the user provided flag value.
// If flag is not set explicitly, it will return 0.
func (id ID) Flag() uint8 {
	if id[5]&0x80 == 0x80 { // not set
		return 0
	}
	return id[5]
}

// Machine returns the 4-byte machine id part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) Machine() []byte {
	return id[5:9]
}

// MachineID returns the uint32 representation of the machine id part.
// It's a runtime error to call this method with an invalid id.
func (id ID) MachineID() uint32 {
	return binary.BigEndian.Uint32(id[5:9])
}

// MachineIP returns the IP representation of the machine id part when
// used with IP/PORT mode.
// It's a runtime error to call this method with an invalid id.
func (id ID) MachineIP() net.IP {
	return net.IPv4(id[5], id[6], id[7], id[8])
}

// Pid returns the process id part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) Pid() uint16 {
	return binary.BigEndian.Uint16(id[9:11])
}

// Port returns the port part of the id when used with IP/PORT mode.
// It's a runtime error to call this method with an invalid id.
func (id ID) Port() uint16 {
	return binary.BigEndian.Uint16(id[9:11])
}

// Counter returns the incrementing value part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) Counter() int32 {
	return int32(binary.BigEndian.Uint32(id[11:15]))
}

// Value implements the driver.Valuer interface.
func (id ID) Value() (driver.Value, error) {
	if id.IsNil() {
		return nil, nil
	}
	b, err := id.MarshalText()
	return string(b), err
}

// Scan implements the sql.Scanner interface.
func (id *ID) Scan(value interface{}) (err error) {
	switch val := value.(type) {
	case string:
		return id.UnmarshalText([]byte(val))
	case []byte:
		return id.UnmarshalText(val)
	case nil:
		*id = nilID
		return nil
	default:
		return fmt.Errorf("xxid: scanning unsupported type: %T", value)
	}
}

// IsNil Returns true if this is a "nil" ID.
func (id ID) IsNil() bool {
	return id == nilID
}

// Bytes returns the byte array representation of `ID`.
func (id ID) Bytes() []byte {
	return id[:]
}

// Short returns the 63 bits int64 representation of the ID consisting of
// timestamp and counter, the first bit is always 0, then the 32 bits
// timestamp, and the 31 bits counter.
func (id ID) Short() int64 {
	t := binary.BigEndian.Uint32(id[0:4])
	c := binary.BigEndian.Uint32(id[11:15])
	return (int64(t) << 31) | int64(c)
}

// NilID returns a zero value for `ID`.
func NilID() ID {
	return nilID
}

// FromString reads an ID from its string representation.
func FromString(id string) (ID, error) {
	x := &ID{}
	err := x.UnmarshalText([]byte(id))
	return *x, err
}

// FromUUID reads an ID from its UUID string representation.
func FromUUID(uuid string) (ID, error) {
	var id ID
	if len(uuid) != uuidLen {
		return id, ErrInvalidID
	}
	text := []byte(uuid)
	copy(text[8:12], uuid[9:13])
	copy(text[12:16], uuid[14:18])
	copy(text[16:20], uuid[19:23])
	copy(text[20:30], uuid[26:36])
	_, err := hex.Decode(text[:], text[:rawLen*2])
	if err != nil {
		return id, ErrInvalidID
	}
	copy(id[:], text)
	return id, nil
}

// FromBytes convert the byte array representation of `ID` back to `ID`.
func FromBytes(b []byte) (ID, error) {
	var id ID
	if len(b) != rawLen || b[rawLen-4]&0x80 != 0 {
		return id, ErrInvalidID
	}
	copy(id[:], b)
	return id, nil
}

// FromShort restore a short int64 id back to a full `ID`.
func FromShort(short int64) (ID, error) {
	return fromShort(defaultGenerator, short)
}

func fromShort(g *Generator, short int64) (ID, error) {
	var id ID
	if short < 0 {
		return id, ErrInvalidID
	}
	// timestamp & counter from short id
	binary.BigEndian.PutUint32(id[:], uint32(short>>31))
	binary.BigEndian.PutUint32(id[11:15], uint32(short)&^(1<<31))
	// flag, machine id and pid from generator
	id[4] = g.flag
	copy(id[5:], g.machineID[:])
	id[9], id[10] = byte(g.pid>>8), byte(g.pid)
	return id, nil
}

// Compare returns an integer comparing two IDs. It behaves just like `bytes.Compare`.
// The result will be 0 if two IDs are identical, -1 if current id is less than the other one,
// and 1 if current id is greater than the other.
func (id ID) Compare(other ID) int {
	return bytes.Compare(id[:], other[:])
}

type sorter []ID

func (s sorter) Len() int           { return len(s) }
func (s sorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sorter) Less(i, j int) bool { return s[i].Compare(s[j]) < 0 }

// Sort sorts an array of IDs in-place.
// It works by wrapping `[]ID` and use `sort.Sort`.
func Sort(ids []ID) {
	sort.Sort(sorter(ids))
}

type byMachine []ID

func (s byMachine) Len() int      { return len(s) }
func (s byMachine) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byMachine) Less(i, j int) bool {
	// machine id and process id
	if x := bytes.Compare(s[i][5:11], s[j][5:11]); x != 0 {
		return x < 0
	}
	// timestamp
	if x := bytes.Compare(s[i][0:4], s[j][0:4]); x != 0 {
		return x < 0
	}
	// counter
	return bytes.Compare(s[i][11:15], s[j][11:15]) < 0
}

// SortByMachine sorts an array of IDs in-place using MachineID and Pid lexicographically.
// It works by wrapping `[]ID` and use `sort.Sort`.
func SortByMachine(ids []ID) {
	sort.Sort(byMachine(ids))
}
