package xxid

const (
	// lexicographic ordering (based on Unicode table) is 0-9A-Za-z
	base62Characters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	offsetUppercase  = 10
	offsetLowercase  = 36
)

// dec is used to convert a base 62 byte into the number value that it represents.
var dec [128]byte

func init() {
	for char := byte(0); char < byte(len(dec)); char++ {
		switch {
		case char >= '0' && char <= '9':
			dec[char] = char - '0'
		case char >= 'A' && char <= 'Z':
			dec[char] = offsetUppercase + char - 'A'
		case char >= 'a' && char <= 'z':
			dec[char] = offsetLowercase + char - 'a'
		default:
			dec[char] = 0xff
		}
	}
}

// encodeBase62 encodes src in binary form to dst in base62 form.
//
// Note that in order to support a couple of optimizations the function
// assumes that:
// 1. the length of dst is exactly you want, unused bytes will be set to '0';
// 2. the length of src is a multiple of 4, else it panics in runtime;
func encodeBase62(dst, src []byte) {
	const uint32base = 1 << 32
	const dstBase = 62

	// Split src into 4 4-byte words, this is where most of the efficiency comes
	// from because this is a O(N^2) algorithm, and we make N = N / 4 by working
	// on 32 bits at a time.
	parts := make([]uint32, 0, len(src)/4)
	for i := 0; i < len(src); i += 4 {
		x := uint32(src[i])<<24 | uint32(src[i+1])<<16 + uint32(src[i+2])<<8 | uint32(src[i+3])
		parts = append(parts, x)
	}

	n := len(dst)
	bp := parts
	bq := [maxBinEncodedLen / 4]uint32{}

	for len(bp) != 0 {
		var value, remainder uint64
		quotient := bq[:0]
		for _, c := range bp {
			value = uint64(c) + remainder*uint32base
			digit := value / dstBase
			remainder = value % dstBase
			if len(quotient) != 0 || digit != 0 {
				quotient = append(quotient, uint32(digit))
			}
		}

		// Writes at the end of the destination buffer because we computed
		// the lowest bits first.
		n--
		dst[n] = base62Characters[remainder]
		bp = quotient
	}

	// Add padding at the head of the destination buffer for all bytes that
	// were not set.
	for i := 0; i < n; i++ {
		dst[i] = '0'
	}
}

// decodeBase62 decodes src in base62 form to dst in binary form.
//
// Note that in order to support a couple of optimizations the function
// assumes that:
// 1. the length of dst is exactly the length of the corresponding binary
// form, else it panics in runtime;
// 2. the length of src is not larger than 38 which is the max possible
// length of an ID in base62 form, else it panics in runtime;
func decodeBase62(dst []byte, src []byte) error {
	const srcBase = 62
	const uint32base = 1 << 32

	parts := make([]byte, 0, maxBase62EncodedLen)
	for _, c := range src {
		x := dec[c&0x7f]
		if x == 0xff {
			return errInvalidBase62Character(c)
		}
		parts = append(parts, x)
	}
	n := len(dst)
	bp := parts
	bq := [38]byte{}

	for len(bp) > 0 {
		var value, remainder uint64
		quotient := bq[:0]
		for _, c := range bp {
			value = uint64(c) + remainder*srcBase
			digit := value / uint32base
			remainder = value % uint32base
			if len(quotient) != 0 || digit != 0 {
				quotient = append(quotient, byte(digit))
			}
		}

		dst[n-4] = byte(remainder >> 24)
		dst[n-3] = byte(remainder >> 16)
		dst[n-2] = byte(remainder >> 8)
		dst[n-1] = byte(remainder)
		n -= 4
		bp = quotient
	}
	return nil
}
