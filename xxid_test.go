package xxid

import (
	"reflect"
	"testing"
)

func TestID_simple(t *testing.T) {
	id := New()
	t.Logf("binary= %v", id.Binary())
	t.Logf("base62= %v", string(id.Base62()))
	t.Logf("string= %v", id.String())
}

func TestIDBinary(t *testing.T) {
	id := New()
	encoded := id.Binary()
	got, err := ParseBinary(encoded)
	if err != nil {
		t.Fatal("failed parse ID from binary representation")
	}
	if got != id {
		t.Fatalf("ParseBinary result not match\n"+
			"src timeMsec= %v, pidOrPort= %v, counter= %v, flag= %v, mIDType= %v, machineID= %v\n"+
			"got timeMsec= %v, pidOrPort= %v, counter= %v, flag= %v, mIDType= %v, machineID= %v",
			id.timeMsec, id.pidOrPort, id.counter, id.flag, id.mIDType, id.machineID,
			got.timeMsec, got.pidOrPort, got.counter, got.flag, got.mIDType, got.machineID)
	}
}

func TestIDBase62(t *testing.T) {
	id := New()
	encoded := id.Base62()
	got, err := ParseBase62(encoded)
	if err != nil {
		t.Fatal("failed parse ID from base62 representation")
	}
	if got != id {
		t.Fatalf("ParseBase62 result not match\n"+
			"src timeMsec= %v, pidOrPort= %v, counter= %v, flag= %v, mIDType= %v, machineID= %v\n"+
			"got timeMsec= %v, pidOrPort= %v, counter= %v, flag= %v, mIDType= %v, machineID= %v",
			id.timeMsec, id.pidOrPort, id.counter, id.flag, id.mIDType, id.machineID,
			got.timeMsec, got.pidOrPort, got.counter, got.flag, got.mIDType, got.machineID)
	}
}

func TestIDString(t *testing.T) {
	id := New()
	encoded := id.String()
	got, err := ParseString(encoded)
	if err != nil {
		t.Fatal("failed parse ID from string representation")
	}
	if got != id {
		t.Fatalf("ParseString result not match\n"+
			"src timeMsec= %v, pidOrPort= %v, counter= %v, flag= %v, mIDType= %v, machineID= %v\n"+
			"got timeMsec= %v, pidOrPort= %v, counter= %v, flag= %v, mIDType= %v, machineID= %v",
			id.timeMsec, id.pidOrPort, id.counter, id.flag, id.mIDType, id.machineID,
			got.timeMsec, got.pidOrPort, got.counter, got.flag, got.mIDType, got.machineID)
	}
}

func TestID_Methods(t *testing.T) {
	id := New()
	table := []struct {
		name string
		want interface{}
		got  interface{}
	}{
		{"Time", id.timeMsec, id.Time().UnixNano() / 1e6},
		{"Pid", id.pidOrPort, id.Pid()},
		{"Port", id.pidOrPort, id.Port()},
		{"Counter", id.counter, id.Counter()},
		{"MachineIDType", id.mIDType, id.MachineIDType()},
		{"MachineID", id.machineID[:4], id.MachineID()},
		{"Flag", uint16(0), id.Flag()},
	}
	for _, tc := range table {
		if !reflect.DeepEqual(tc.want, tc.got) {
			t.Fatalf("id.%s() result not match, want= %v, got= %v", tc.name, tc.want, tc.got)
		}
	}
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New()
	}
}

func BenchmarkID_Binary(b *testing.B) {
	id := New()
	for i := 0; i < b.N; i++ {
		_ = id.Binary()
	}
}

func BenchmarkID_Base62(b *testing.B) {
	id := New()
	for i := 0; i < b.N; i++ {
		_ = id.Base62()
	}
}

func BenchmarkID_String(b *testing.B) {
	id := New()
	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}

func BenchmarkParseBinary(b *testing.B) {
	buf := New().Binary()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBinary(buf)
	}
}

func BenchmarkParseBase62(b *testing.B) {
	b62 := New().Base62()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBase62(b62)
	}
}

func BenchmarkParseString(b *testing.B) {
	str := New().String()
	for i := 0; i < b.N; i++ {
		_, _ = ParseString(str)
	}
}
