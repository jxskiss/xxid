package xxid

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New()
		}
	})
}

func BenchmarkNewString(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New().String()
		}
	})
}

func BenchmarkFromString(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = FromString("HiR5qKopPNzBU8s7lW0H")
		}
	})
}

//func BenchmarkUUIDv1(b *testing.B) {
//	b.RunParallel(func(pb *testing.PB) {
//		for pb.Next() {
//			_ = uuid.NewV1().String()
//		}
//	})
//}

//func BenchmarkUUIDv4(b *testing.B) {
//	b.RunParallel(func(pb *testing.PB) {
//		for pb.Next() {
//			_ = uuid.NewV4().String()
//		}
//	})
//}

func Benchmark_xxid_Encode(b *testing.B) {
	id := New()

	for i := 0; i < b.N; i++ {
		id.String()
	}
}

func Benchmark_xxid_Decode(b *testing.B) {
	id := New()
	str := id.String()

	for i := 0; i < b.N; i++ {
		FromString(str)
	}
}

//func Benchmark_xid_Encode(b *testing.B) {
//	id := xid.New()
//
//	for i := 0; i < b.N; i++ {
//		id.String()
//	}
//}

//func Benchmark_xid_Decode(b *testing.B) {
//	id := xid.New()
//	str := id.String()
//
//	for i := 0; i < b.N; i++ {
//		xid.FromString(str)
//	}
//}

//func Benchmark_ksuid_Encode(b *testing.B) {
//	id := ksuid.New()
//
//	for i := 0; i < b.N; i++ {
//		id.String()
//	}
//}

//func Benchmark_ksuid_Decode(b *testing.B) {
//	id := ksuid.New()
//	str := id.String()
//
//	for i := 0; i < b.N; i++ {
//		ksuid.Parse(str)
//	}
//}
