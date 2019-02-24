package xxid

const (
	// lexicographic ordering (based on Unicode table) is 0-9A-Za-z
	base62Characters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	offsetUppercase  = 10
	offsetLowercase  = 36
)

var (
	// dec is used to convert a base 62 byte into the number value that it represents.
	dec [128]byte

	// A string-encoded minimum value for an ID
	minEncoded = []byte("00000000000000000000")
	// A string-encoded maximum value for an ID
	maxEncoded = []byte("wUlFeUHcE1B9u5BFOMyF")
)

func init() {
	for char := byte(0); char < byte(len(dec)); char++ {
		switch {
		case char >= '0' && char <= '9':
			dec[char] = char - '0'
		case char >= 'A' && char <= 'Z':
			dec[char] = offsetUppercase + char - 'A'
		case char >= 'a' && char <= 'z':
			dec[char] = offsetLowercase + char - 'a'
		}
	}
}

// This function encodes the base 62 representation of the src ID in binary
// form into dst.
//
// In order to support a couple of optimizations the function assumes that src
// is 15 bytes long and dst is 20 bytes long.
//
// Any unused bytes in dst will be set to the padding '0' byte.
func encodeBase62(dst []byte, src []byte) {
	const uint32base = 1 << 32
	const bits23base = 1 << 23
	const dstBase = 62

	// Split src into 4 4-byte words, this is where most of the efficiency comes
	// from because this is a O(N^2) algorithm, and we make N = N / 4 by working
	// on 32 bits at a time.
	parts := [4]uint32{
		uint32(src[0])<<24 | uint32(src[1])<<16 | uint32(src[2])<<8 | uint32(src[3]),
		uint32(src[4])<<24 | uint32(src[5])<<16 | uint32(src[6])<<8 | uint32(src[7]),
		uint32(src[8])<<24 | uint32(src[9])<<16 | uint32(src[10])<<8 |
			uint32(src[11]<<1) | uint32(src[12]>>7),
		(uint32(src[12])<<16 | uint32(src[13])<<8 | uint32(src[14])) & 0x7fffff,
	}
	n := len(dst)
	bp := parts[:]
	bq := [4]uint32{}

	for len(bp) != 0 {
		quotient := bq[:0]
		remainder := uint64(0)
		value := uint64(0)

		for i, c := range bp {
			if i == len(bp)-1 { // the low three bytes (31 bits)
				value = uint64(c) + uint64(remainder)*bits23base
			} else {
				value = uint64(c) + uint64(remainder)*uint32base
			}
			digit := value / dstBase
			remainder = value % dstBase

			if len(quotient) != 0 || digit != 0 {
				quotient = append(quotient, uint32(digit))
			}
		}

		// Writes at the end of the destination buffer because we computed the
		// lowest bits first.
		n--
		dst[n] = base62Characters[remainder]
		bp = quotient
	}

	// Add padding at the head of the destination buffer for all bytes that were
	// not set.
	copy(dst[:n], minEncoded)
}

// This function decodes the base 62 representation of the src ID to the
// binary form into dst.
//
// In order to support a couple of optimizations the function assumes that src
// is 20 bytes long and dst is 15 bytes long.
//
// Any unused bytes in dst will be set to zero.
func decodeBase62(dst []byte, src []byte) error {
	const srcBase = 62
	const uint32base = 1 << 32
	const bits23base = 1 << 23

	// This line helps BCE (Bounds Check Elimination).
	// It may be safely removed.
	_ = src[encodedLen-1]

	parts := [encodedLen]byte{
		dec[src[0]],
		dec[src[1]],
		dec[src[2]],
		dec[src[3]],
		dec[src[4]],
		dec[src[5]],
		dec[src[6]],
		dec[src[7]],
		dec[src[8]],
		dec[src[9]],

		dec[src[10]],
		dec[src[11]],
		dec[src[12]],
		dec[src[13]],
		dec[src[14]],
		dec[src[15]],
		dec[src[16]],
		dec[src[17]],
		dec[src[18]],
		dec[src[19]],
	}
	n := len(dst)
	bp := parts[:]
	bq := [encodedLen]byte{}

	for len(bp) > 0 {
		quotient := bq[:0]
		remainder := uint64(0)
		if n == rawLen { // the low three bytes (31 bits)
			dstBase := uint64(bits23base)
			for _, c := range bp {
				value := uint64(c) + uint64(remainder)*srcBase
				digit := value / dstBase
				remainder = value % dstBase

				if len(quotient) != 0 || digit != 0 {
					quotient = append(quotient, byte(digit))
				}
			}
			dst[n-3] = byte(remainder >> 16)
			dst[n-2] = byte(remainder >> 8)
			dst[n-1] = byte(remainder)
			n -= 3
		} else {
			dstBase := uint64(uint32base)
			for _, c := range bp {
				value := uint64(c) + uint64(remainder)*srcBase
				digit := value / dstBase
				remainder = value % dstBase

				if len(quotient) != 0 || digit != 0 {
					quotient = append(quotient, byte(digit))
				}
			}
			dst[n-4] = byte(remainder >> 24)
			dst[n-3] = byte(remainder >> 16)
			dst[n-2] = byte(remainder >> 8)
			dst[n-1] = byte(remainder)
			n -= 4
		}
		bp = quotient
	}

	var zero [20]byte
	copy(dst[:n], zero[:])
	if dst[11]&0x1 == 0x1 {
		dst[12] |= 0x80
	}
	dst[11] >>= 1
	return nil
}
