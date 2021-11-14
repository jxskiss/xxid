package xxid

import (
	"bytes"
	cryptorand "crypto/rand"
	"math/rand"
	"testing"
)

func Test_encodeBase62_decodeBase62(t *testing.T) {
	binLengthList := []int{16, 20, 28}
	b62LengthList := []int{22, 27, 38}

	for i, binLen := range binLengthList {
		b62Len := b62LengthList[i]

		zeroSrc := make([]byte, binLen)
		zeroEncoded := make([]byte, b62Len)
		encodeBase62(zeroEncoded, zeroSrc)
		zeroDecoded := make([]byte, binLen)
		err := decodeBase62(zeroDecoded, zeroEncoded)
		if err != nil {
			t.Fatalf("failed decode zero bytes, binLen= %v, err= %v", binLen, err)
		}
		if !bytes.Equal(zeroSrc, zeroDecoded) {
			t.Fatalf("decoded zero bytes not match, binLen= %v, decoded= %v", binLen, zeroDecoded)
		}

		ffSrc := make([]byte, binLen)
		for i := range ffSrc {
			ffSrc[i] = 0xff
		}
		ffEncoded := make([]byte, b62Len)
		encodeBase62(ffEncoded, ffSrc)
		ffDecoded := make([]byte, binLen)
		err = decodeBase62(ffDecoded, ffEncoded)
		if err != nil {
			t.Fatalf("failed decode 0xff bytes, binLen= %v, err= %v", binLen, err)
		}
		if !bytes.Equal(ffSrc, ffDecoded) {
			t.Fatalf("decoded 0xff bytes not match, binLen= %v, decoded= %v", binLen, ffDecoded)
		}
	}

	for i := 0; i < 1000; i++ {
		n := rand.Intn(len(binLengthList))
		binLen := binLengthList[n]
		b62Len := b62LengthList[n]

		src := make([]byte, binLen)
		_, err := cryptorand.Read(src)
		if err != nil {
			panic(err)
		}

		encoded := make([]byte, b62Len)
		encodeBase62(encoded, src)
		decoded := make([]byte, binLen)
		err = decodeBase62(decoded, encoded)
		if err != nil {
			t.Fatalf("failed decode random bytes, binLen= %v, src= %v, encoded= %v, err= %v",
				binLen, src, encoded, err)
		}
		if !bytes.Equal(src, decoded) {
			t.Fatalf("decoded random bytes not match, binLen= %v, src= %v, encoded= %v, decoded= %v",
				binLen, src, encoded, decoded)
		}
	}
}
