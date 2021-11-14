package xxid

import (
	"crypto/md5"
	"hash/crc32"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jxskiss/xxid/v2/machineid"
)

var (
	defaultGenerator *Generator
	counter          uint32
)

func init() {
	machineID, mIDType := readMachineID()
	pid := readProcessID()
	counter = runtime_fastrand()
	defaultGenerator = &Generator{
		mIDType:   mIDType,
		pidOrPort: pid,
	}
	copy(defaultGenerator.machineID[:4], machineID[:])
}

// A Generator holds some machine information which is used to generate
// unique IDs. Some information can be configured by user.
type Generator struct {
	mIDType   MachineIDType
	machineID [16]byte
	pidOrPort uint16
	flag      uint16
}

// NewGenerator makes a new generator initialized with same machineID and
// pid as the default generator, this is useful to specify machine ID, IP,
// port and flag value instead of the defaults.
//
// For general purpose without configuring machine ID, IP, port or flag,
// New and NewWithTime are recommended in most cases.
func NewGenerator() *Generator {
	gen := &Generator{
		mIDType:   defaultGenerator.mIDType,
		machineID: defaultGenerator.machineID,
		pidOrPort: defaultGenerator.pidOrPort,
	}
	return gen
}

// UseMachineID changes the machine ID of the generator to user
// specified bytes.
//
// Length of the provided bytes must be 4, 8 or 16, else it panics,
// the corresponding MachineIDType will be Specified4, Specified8
// or Specified16.
func (g *Generator) UseMachineID(id []byte) *Generator {
	switch len(id) {
	case 4:
		g.mIDType = Specified4
		copy(g.machineID[:4], id)
	case 8:
		g.mIDType = Specified8
		copy(g.machineID[:8], id)
	case 16:
		g.mIDType = Specified16
		copy(g.machineID[:16], id)
	default:
		panic(errUnsupportedMachineIDLength)
	}
	return g
}

// UseIPv4 sets the generator to use the given IP v4 as machine ID.
func (g *Generator) UseIPv4(ip net.IP) *Generator {
	g.mIDType = IPv4
	copy(g.machineID[:4], ip.To4())
	return g
}

// UseIPv6 sets the generator to use the given IP v6 as machine ID.
func (g *Generator) UseIPv6(ip net.IP) *Generator {
	g.mIDType = IPv6
	copy(g.machineID[:16], ip.To16())
	return g
}

// UsePort sets the generator to use the given port number.
func (g *Generator) UsePort(port uint16) *Generator {
	if port > 0 {
		g.pidOrPort = port
	}
	return g
}

// UseFlag sets the generator to use the given flag.
//
// Note that only 15 bits are allowed for flag, if the highest bit is set,
// it will be discarded.
func (g *Generator) UseFlag(flag uint16) *Generator {
	g.flag = flag | flagMask
	return g
}

// New generates a unique ID.
func (g *Generator) New() ID {
	timeMsec, incr := readTimeAndCounter()
	return newID(g, timeMsec, incr)
}

// NewWithTime generates an ID with the given time.
func (g *Generator) NewWithTime(t time.Time) ID {
	timeMsec := t.UnixNano() / 1e6
	incr := incrCounter()
	return newID(g, timeMsec, incr)
}

// readMachineID reads machine ID from the host operating system.
// If it fails to get machine ID from the host, it returns a random value.
func readMachineID() ([4]byte, MachineIDType) {
	var id [4]byte
	hid, err := machineid.ID()
	if err != nil || len(hid) == 0 {
		hid, err = os.Hostname()
	}
	if err == nil && len(hid) != 0 {
		hw := md5.New()
		hw.Write([]byte(hid))
		copy(id[:], hw.Sum(nil))
		return id, HostID
	}

	// Fallback to rand number if machine id can't be gathered.
	x := runtime_fastrand()
	id[0] = byte(x >> 24)
	id[1] = byte(x >> 16)
	id[2] = byte(x >> 8)
	id[3] = byte(x)
	return id, Random
}

func readProcessID() uint16 {
	pid := uint16(os.Getpid())
	// If /proc/self/cpuset exists and is not /, we can assume that we are in a
	// form of container and use the content of cpuset xor-ed with the PID in
	// order to get a reasonable machine global unique PID.
	b, err := ioutil.ReadFile("/proc/self/cpuset")
	if err == nil && len(b) > 1 {
		pid ^= uint16(crc32.ChecksumIEEE(b))
	}
	return pid
}

func randFlag() uint16 {
	return uint16(runtime_fastrand() >> 17)
}

func incrCounter() uint16 {
	return uint16(atomic.AddUint32(&counter, 1))
}

var (
	incrMu         sync.Mutex
	timeAndCounter int64
)

// readTimeAndCounter guarantees that the combination of the returned
// time and counter will never be duplicate inside a process, even the
// clock has been turned back or leap second happens.
func readTimeAndCounter() (timeMsec int64, counter uint16) {
	t := time.Now().UnixNano() / 1e6
	c := incrCounter()
	tac := t<<16 | int64(c) // time and counter

	incrMu.Lock()
	prev := timeAndCounter
	if tac <= prev {
		tac = prev + 1
		t, c = tac>>16, uint16(tac)
	}
	timeAndCounter = tac
	incrMu.Unlock()
	return t, c
}
