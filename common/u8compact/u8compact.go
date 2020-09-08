package u8compact

import (
	"math/big"

	"github.com/tsfdsong/go-polkadot/common/bnutil"
	"github.com/tsfdsong/go-polkadot/common/mathutil"
	"github.com/tsfdsong/go-polkadot/common/u8util"
)

// DefaultBitLength ...
const DefaultBitLength = 32

// MaxU8 ...
var MaxU8 = new(big.Int).Sub(mathutil.Pow(big.NewInt(2), big.NewInt(8-2)), big.NewInt(1))

// MaxU16 ...
var MaxU16 = new(big.Int).Sub(mathutil.Pow(big.NewInt(2), big.NewInt(16-2)), big.NewInt(1))

// MaxU32 ...
var MaxU32 = new(big.Int).Sub(mathutil.Pow(big.NewInt(2), big.NewInt(32-2)), big.NewInt(1))

// FromUint8Slice retrieves the offset and encoded length from a compact-prefixed value
func FromUint8Slice(input []uint8, bitLength int) (int, *big.Int) {
	if len(input) == 0 {
		return 0, nil
	}

	flag := input[0] & 0x3

	if flag == 0x0 {
		x := big.NewInt(int64(input[0]))
		return 1, new(big.Int).Rsh(x, 2)
	} else if flag == 0x1 {
		x := bnutil.ToBN(input[0:2], true)
		y := new(big.Int).Rsh(x, 2)
		return 2, y
	} else if flag == 0x2 {
		x := bnutil.ToBN(input[0:4], true)
		return 4, new(big.Int).Rsh(x, 2)
	}

	offset := 1 + (bitLength / 8)
	end := offset
	if end > len(input) {
		end = len(input)
	}

	return offset, bnutil.ToBN(input[1:end], true)
}

// CompactToUint8Slice encodes a number into a compact representation.
func CompactToUint8Slice(value *big.Int, bitLength int) []uint8 {
	if value.Cmp(MaxU8) <= 0 {
		return []uint8{uint8(value.Int64() << 2)}
	} else if value.Cmp(MaxU16) <= 0 {
		i := new(big.Int).Add(new(big.Int).Lsh(value, 2), big.NewInt(1))
		return bnutil.ToUint8Slice(i, 16, true, false)
	} else if value.Cmp(MaxU32) <= 0 {
		i := new(big.Int).Add(new(big.Int).Lsh(value, 2), big.NewInt(2))
		return bnutil.ToUint8Slice(i, 32, true, false)
	}

	slice := bnutil.ToUint8Slice(value, -1, true, false)
	length := len(slice)

	// adjust to the minimum number of bytes
	for slice[length-1] == 0 {
		length--
	}

	if length < 4 {
		panic("previous tests match anything less than 2^30; qed")
	}

	return u8util.Concat(
		// subtract 4 as minimum (also catered for in decoding)
		[]uint8{uint8(((length - 4) << 2) + 0x3)}, // NOTE: 0x3 = 0b11
		slice[0:length],
	)
}

// AddLength adds a length prefix to the input value.
func AddLength(input []uint8, bitLength int) []uint8 {
	return u8util.Concat(
		CompactToUint8Slice(big.NewInt(int64(len(input))), bitLength),
		input,
	)
}

// StripLength removes the length prefix, returning both the total length (including the value + compact encoding) and the decoded value with the correct length.
func StripLength(input []uint8, bitLength int) (int, []uint8) {
	if len(input) == 0 {
		return 0, []uint8{}
	}

	offset, length := FromUint8Slice(input, bitLength)
	total := offset + int(length.Uint64())

	return total, input[offset:total]
}
