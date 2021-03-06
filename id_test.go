package xxid

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

type IDParts struct {
	id        ID
	timestamp int64
	machine   []byte
	pid       uint16
	counter   int32
}

var IDs = []IDParts{
	IDParts{
		ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9},
		1300816219,
		[]byte{0x7c, 0x60, 0xf4, 0x86},
		0xe428,
		4271561,
	},
	IDParts{
		ID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		0,
		[]byte{0x00, 0x00, 0x00, 0x00},
		0x0000,
		0,
	},
	IDParts{
		ID{0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x00, 0x00, 0x01},
		0,
		[]byte{0xaa, 0xbb, 0xcc, 0xdd},
		0xeeff,
		1,
	},
}

func TestIDPartsExtraction(t *testing.T) {
	for i, v := range IDs {
		t.Run(fmt.Sprintf("Test%d", i), func(t *testing.T) {
			if got, want := v.id.Time(), time.Unix(v.timestamp+epoch, 0); got != want {
				t.Errorf("Time() = %v, want %v", got, want)
			}
			if got, want := v.id.Machine(), v.machine; !bytes.Equal(got, want) {
				t.Errorf("Machine() = %v, want %v", got, want)
			}
			if got, want := v.id.Pid(), v.pid; got != want {
				t.Errorf("Pid() = %v, want %v", got, want)
			}
			if got, want := v.id.Counter(), v.counter; got != want {
				t.Errorf("Counter() = %v, want %v", got, want)
			}
		})
	}
}

func TestGenerator(t *testing.T) {
	flag := uint8(123)
	ip := net.ParseIP("10.9.8.7")
	port := uint16(65432)
	gen := NewGenerator().UseFlag(flag).UseIP(ip).UsePort(port)

	id := gen.New()
	if got, want := id.Flag(), flag; got != want {
		t.Errorf("Flag() = %v, want %v", got, want)
	}
	if got, want := id.MachineIP(), ip; got.String() != want.String() {
		t.Errorf("MachineIP() = %v, want %v", got, want)
	}
	if got, want := id.Port(), port; got != want {
		t.Errorf("Port() = %v, want %v", got, want)
	}
}

func TestNew(t *testing.T) {
	// Generate 10 ids
	ids := make([]ID, 10)
	for i := 0; i < 10; i++ {
		ids[i] = New()
	}
	for i := 1; i < 10; i++ {
		prevID := ids[i-1]
		id := ids[i]
		// Test for uniqueness among all other 9 generated ids
		for j, tid := range ids {
			if j != i {
				if id.Compare(tid) == 0 {
					t.Errorf("generated ID is not unique (%d/%d)", i, j)
				}
			}
		}
		// Check that timestamp was incremented and is within 30 seconds of the previous one
		secs := id.Time().Sub(prevID.Time()).Seconds()
		if secs < 0 || secs > 30 {
			t.Error("wrong timestamp in generated ID")
		}
		// Check that machine ids are the same
		if !bytes.Equal(id.Machine(), prevID.Machine()) {
			t.Error("machine ID not equal")
		}
		// Check that pids are the same
		if id.Pid() != prevID.Pid() {
			t.Error("pid not equal")
		}
		// Test for proper increment
		if got, want := int(id.Counter()-prevID.Counter()), 1; got != want {
			t.Errorf("wrong increment in generated ID, delta=%v, want %v", got, want)
		}
	}
}

func TestIDString(t *testing.T) {
	id := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	if got, want := id.String(), "HiR5qKopPNzBU8s7lW0H"; got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestIDAddr(t *testing.T) {
	g := NewGenerator().UseIP(net.ParseIP("10.1.2.3")).UsePort(12345)
	id := g.New()
	if got, want := id.Addr(), "10.1.2.3:12345"; got != want {
		t.Errorf("Addr() = %v, want = %v", got, want)
	}
}

func TestFromString(t *testing.T) {
	got, err := FromString("HiR5qKopPNzBU8s7lW0H")
	if err != nil {
		t.Fatal(err)
	}
	want := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	if got != want {
		t.Errorf("FromString() = %v, want %v", got, want)
	}
}

func TestFromStringInvalid(t *testing.T) {
	_, err := FromString("invalid")
	if err != ErrInvalidID {
		t.Errorf("FromString(invalid) err=%v, want %v", err, ErrInvalidID)
	}
}

type jsonType struct {
	ID  *ID
	Str string
}

func TestIDJSONMarshaling(t *testing.T) {
	id := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	v := jsonType{ID: &id, Str: "test"}
	data, err := json.Marshal(&v)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"ID":"HiR5qKopPNzBU8s7lW0H","Str":"test"}`; got != want {
		t.Errorf("json.Marshal() = %v, want %v", got, want)
	}
}

func TestIDJSONUnmarshaling(t *testing.T) {
	data := []byte(`{"ID":"HiR5qKopPNzBU8s7lW0H","Str":"test"}`)
	v := jsonType{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		t.Fatal(err)
	}
	want := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	if got := *v.ID; got.Compare(want) != 0 {
		t.Errorf("json.Unmarshal() = %v, want %v", got, want)
	}

}

func TestIDJSONUnmarshalingError(t *testing.T) {
	v := jsonType{}
	err := json.Unmarshal([]byte(`{"ID":"xM4E2MR0UI3E8A215N4G"}`), &v)
	if err != ErrInvalidID {
		t.Errorf("json.Unmarshal() err=%v, want %v", err, ErrInvalidID)
	}
	err = json.Unmarshal([]byte(`{"ID":"TYjhW2D0huQoQS"}`), &v)
	if err != ErrInvalidID {
		t.Errorf("json.Unmarshal() err=%v, want %v", err, ErrInvalidID)
	}
	err = json.Unmarshal([]byte(`{"ID":"TYjhW2D0huQoQS3kdk"}`), &v)
	if err != ErrInvalidID {
		t.Errorf("json.Unmarshal() err=%v, want %v", err, ErrInvalidID)
	}
}

func TestIDDriverValue(t *testing.T) {
	id := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	got, err := id.Value()
	if err != nil {
		t.Fatal(err)
	}
	if want := "HiR5qKopPNzBU8s7lW0H"; got != want {
		t.Errorf("Value() = %v, want %v", got, want)
	}
}

func TestIDDriverScan(t *testing.T) {
	got := ID{}
	err := got.Scan("HiR5qKopPNzBU8s7lW0H")
	if err != nil {
		t.Fatal(err)
	}
	want := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	if got.Compare(want) != 0 {
		t.Errorf("Scan() = %v, want %v", got, want)
	}
}

func TestIDDriverScanError(t *testing.T) {
	id := ID{}
	if got, want := id.Scan(0), errors.New("xxid: scanning unsupported type: int"); !reflect.DeepEqual(got, want) {
		t.Errorf("Scan() err=%v, want %v", got, want)
	}
	if got, want := id.Scan("0"), ErrInvalidID; got != want {
		t.Errorf("Scan() err=%v, want %v", got, want)
	}
}

func TestIDDriverScanByteFromDatabase(t *testing.T) {
	got := ID{}
	bs := []byte("HiR5qKopPNzBU8s7lW0H")
	err := got.Scan(bs)
	if err != nil {
		t.Fatal(err)
	}
	want := ID{0x4d, 0x88, 0xe1, 0x5b, 0x00, 0x7c, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x00, 0x41, 0x2d, 0xc9}
	if got.Compare(want) != 0 {
		t.Errorf("Scan() = %v, want %v", got, want)
	}
}

func TestID_IsNil(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want bool
	}{
		{
			name: "ID not nil",
			id:   New(),
			want: false,
		},
		{
			name: "Nil ID",
			id:   ID{},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got, want := tt.id.IsNil(), tt.want; got != want {
				t.Errorf("IsNil() = %v, want %v", got, want)
			}
		})
	}
}

func TestNilID(t *testing.T) {
	got := ID{}
	if want := NilID(); !reflect.DeepEqual(got, want) {
		t.Error("NilID() not equal ID{}")
	}
}

func TestNilID_IsNil(t *testing.T) {
	if !NilID().IsNil() {
		t.Error("NilID().IsNil() is not true")
	}
}

func TestFromBytes_Invariant(t *testing.T) {
	want := New()
	got, err := FromBytes(want.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got.Compare(want) != 0 {
		t.Error("FromBytes(id.Bytes()) != id")
	}
}

func TestFromBytes_InvalidBytes(t *testing.T) {
	cases := []struct {
		length     int
		shouldFail bool
	}{
		{11, true},
		{12, true},
		{13, true},
		{14, true},
		{15, false},
		{16, true},
	}
	for _, c := range cases {
		b := make([]byte, c.length, c.length)
		_, err := FromBytes(b)
		if got, want := err != nil, c.shouldFail; got != want {
			t.Errorf("FromBytes() error got %v, want %v", got, want)
		}
	}
}

func TestFromUUID_Invariant(t *testing.T) {
	want := New()
	got, err := FromUUID(want.UUID())
	if err != nil {
		t.Fatal(err)
	}
	if got.Compare(want) != 0 {
		t.Error("FromUUID(id.UUID()) != id")
	}
}

func TestFromUUID_Validation(t *testing.T) {
	var uuid string

	// valid smallest uuid
	uuid = "00000000-0000-0000-0000-000000000000"
	got, err := FromUUID(uuid)
	if err != nil {
		t.Errorf("FromUUID() got unexpected err = %v", err)
	}

	// valid largest uuid
	uuid = "ffffffff-ffff-ffff-ffff-00ff7fffffff"
	got, err = FromUUID(uuid)
	if err != nil {
		t.Errorf("FromUUID() got unexpected error = %v", err)
	}
	if got.String() != string(maxEncoded) {
		t.Errorf("FromUUID() got wrong max value, got = %v, want = %v", got, string(maxEncoded))
	}

	// invalid uuid: unused chars
	uuid = "00000000-0000-0000-0000-010000000000"
	got, err = FromUUID(uuid)
	if err != ErrInvalidID {
		t.Errorf("FromUUID() error got %v, want %v", err, ErrInvalidID)
	}

	// invalid uuid: counter greater than max possible value
	uuid = "00000000-0000-0000-0000-000080000000"
	got, err = FromUUID(uuid)
	if err != ErrInvalidID {
		t.Errorf("FromUUID() error got %v, want %v", err, ErrInvalidID)
	}
}

func TestFromShort(t *testing.T) {
	want := New()
	got, err := FromShort(want.Short())
	if err != nil {
		t.Fatal(err)
	}
	if got.Compare(want) != 0 {
		t.Error("FromShort(id.Short()) != id")
	}
}

func TestFromShort_Invalid(t *testing.T) {
	cases := []struct {
		short      int64
		shouldFail bool
	}{
		{-2, true},
		{-1, true},
		{0, false},
		{1, false},
	}
	for _, c := range cases {
		_, err := FromShort(c.short)
		if got, want := err != nil, c.shouldFail; got != want {
			t.Errorf("FromShort() error got %v, want %v", got, want)
		}
	}
}

func TestID_Compare(t *testing.T) {
	pairs := []struct {
		left     ID
		right    ID
		expected int
	}{
		{IDs[1].id, IDs[0].id, -1},
		{ID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, IDs[2].id, -1},
		{IDs[0].id, IDs[0].id, 0},
	}
	for _, p := range pairs {
		if p.expected != p.left.Compare(p.right) {
			t.Errorf("%s Compare to %s should return %d", p.left, p.right, p.expected)
		}
		if -1*p.expected != p.right.Compare(p.left) {
			t.Errorf("%s Compare to %s should return %d", p.right, p.left, -1*p.expected)
		}
	}
}

var IDList = []ID{IDs[0].id, IDs[1].id, IDs[2].id}

func TestSorter_Len(t *testing.T) {
	if got, want := sorter([]ID{}).Len(), 0; got != want {
		t.Errorf("Len() %v, want %v", got, want)
	}
	if got, want := sorter(IDList).Len(), 3; got != want {
		t.Errorf("Len() %v, want %v", got, want)
	}
}

func TestSorter_Less(t *testing.T) {
	sorter := sorter(IDList)
	if !sorter.Less(1, 0) {
		t.Errorf("Less(1, 0) not true")
	}
	if sorter.Less(2, 1) {
		t.Errorf("Less(2, 1) true")
	}
	if sorter.Less(0, 0) {
		t.Errorf("Less(0, 0) true")
	}
}

func TestSorter_Swap(t *testing.T) {
	ids := make([]ID, 0)
	ids = append(ids, IDList...)
	sorter := sorter(ids)
	sorter.Swap(0, 1)
	if got, want := ids[0], IDList[1]; !reflect.DeepEqual(got, want) {
		t.Error("ids[0] != IDList[1]")
	}
	if got, want := ids[1], IDList[0]; !reflect.DeepEqual(got, want) {
		t.Error("ids[1] != IDList[0]")
	}
	sorter.Swap(2, 2)
	if got, want := ids[2], IDList[2]; !reflect.DeepEqual(got, want) {
		t.Error("ids[2], IDList[2]")
	}
}

func TestSort(t *testing.T) {
	ids := make([]ID, 0)
	ids = append(ids, IDList...)
	Sort(ids)
	if got, want := ids, []ID{IDList[1], IDList[2], IDList[0]}; !reflect.DeepEqual(got, want) {
		t.Fail()
	}
}

func TestSortByMachine(t *testing.T) {
	g1 := NewGenerator().UseIP(net.ParseIP("10.0.0.1")).UsePort(1234)
	g2 := NewGenerator().UseIP(net.ParseIP("10.0.0.1")).UsePort(2345)
	g3 := NewGenerator().UseIP(net.ParseIP("10.0.0.2")).UsePort(1234)
	g4 := NewGenerator().UseIP(net.ParseIP("10.0.0.2")).UsePort(2345)

	id1, id2, id3, id4 := g1.New(), g2.New(), g3.New(), g4.New()
	ids := []ID{id4, id3, id2, id1}
	SortByMachine(ids)
	if got, want := ids, []ID{id1, id2, id3, id4}; !reflect.DeepEqual(got, want) {
		t.Fail()
	}
}
