package triehash

import "github.com/tsfdsong/go-polkadot/common/hexutil"

// helper
func hexToU8a(s string) []uint8 {
	u8, err := hexutil.ToUint8Slice(s, -1)
	if err != nil {
		panic(err)
	}

	return u8
}
