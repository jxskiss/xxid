package xxid

import (
	"bytes"
	"net"
	"testing"
)

func TestGenerator(t *testing.T) {
	gen := NewGenerator()
	if !bytes.Equal(New().MachineID(), gen.New().MachineID()) {
		t.Fatalf("default generator MachineID not match")
	}
	if New().Pid() != gen.New().Pid() {
		t.Fatalf("default generator Pid not match")
	}
	if New().Port() != gen.New().Pid() {
		t.Fatalf("default generator Port not match")
	}

	machineID := []byte{8, 7, 6, 5, 4, 3, 2, 1}
	gen = NewGenerator().UseMachineID(machineID)
	if !bytes.Equal(machineID, gen.New().MachineID()) {
		t.Fatalf("custom generator MachineID not match")
	}

	port := 9876
	ipV4 := net.ParseIP("10.9.8.7")
	gen = NewGenerator().UseIPv4(ipV4).UsePort(uint16(port))
	if !ipV4.Equal(gen.New().IP()) {
		t.Fatalf("custom IPv4 machine ID not match")
	}
	if got := gen.New().IPPortAddr(); got != "10.9.8.7:9876" {
		t.Fatalf("IPv4 IP port address not match, got= %v", got)
	}

	ipV6 := net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	gen = NewGenerator().UseIPv6(ipV6).UsePort(uint16(port))
	if !ipV6.Equal(gen.New().IP()) {
		t.Fatalf("custom IPv6 machine ID not match")
	}
	if got := gen.New().IPPortAddr(); got != "[2001:db8:85a3::8a2e:370:7334]:9876" {
		t.Fatalf("IPv6 IP port address not match, got= %v", got)
	}

	flag := 12345
	gen = NewGenerator().UseFlag(uint16(flag))
	if gen.New().Flag() != uint16(flag) {
		t.Fatalf("custom flag value not match")
	}
}

func Benchmark_readTimeAndCounter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = readTimeAndCounter()
	}
}
