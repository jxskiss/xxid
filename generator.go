package xxid

import (
	"crypto/rand"
	"fmt"
	"net"
	"time"
)

type Generator struct {
	// machineID stores machine id generated once and used in subsequent calls
	// to the New() function. When UseIP() is called, the provided IPv4 address
	// will be stored as machineID.
	machineID [4]byte
	// pid stores the current process id. When UsePort() is called, the provided
	// port will be cast to uint16 and stored as pid.
	pid uint16
	// flag can be used to store user defined flag with 7 valid bits (0 - 127),
	// the default generator's flag will be filled with random bits. User can
	// use this to distinct different IDC, business or whatever they want.
	flag uint8
	// counter is atomically incremented when generating a new ID using the
	// New() function. It's used as a counter part of an id.
	// This id will be initialized with a random value.
	counter uint32
}

// NewGenerator makes a new generator initialized with same machineID and pid
// as the package's default generator, and new random flag and counter, this
// is useful to specify IP, port and flag instead of the default machineID,
// process id and random flag.
// For general purpose without provided IP, port or flag, please just use the
// package's default functions `New` and `NewWithTime`.
func NewGenerator() *Generator {
	var b = make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("xxid: cannot generate random number: %v", err))
	}
	g := &Generator{
		machineID: defaultGenerator.machineID,
		pid:       defaultGenerator.pid,
	}
	// Always mark random flag's highest bit to distinct from user specified flag.
	g.flag = b[0] | 0x80
	g.counter = uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return g
}

func (g *Generator) UseIP(ip net.IP) *Generator {
	ip = ip.To4()
	if ip != nil && !ip.Equal(net.IPv4zero) && !ip.IsMulticast() && !ip.IsLoopback() {
		copy(g.machineID[:], ip)
	}
	return g
}

func (g *Generator) UsePort(port int) *Generator {
	if port > 0 {
		g.pid = uint16(port)
	}
	return g
}

func (g *Generator) UseFlag(flag uint8) *Generator {
	if flag >= 0x80 {
		panic("xxid: invalid flag value out of range 0-127")
	}
	g.flag = flag
	return g
}

// New generates a globally unique ID.
func (g *Generator) New() ID {
	return newID(g, time.Now())
}

// NewWithTime generates a globally unique ID with the given time.
func (g *Generator) NewWithTime(t time.Time) ID {
	return newID(g, t)
}

// FromShort restore a short int64 id to a full unique ID.
func (g *Generator) FromShort(short int64) (ID, error) {
	return fromShort(g, short)
}
