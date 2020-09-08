package triedb

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/tsfdsong/go-polkadot/common/triehash"
)

func TestSnapshots(t *testing.T) {
	codec := NewTrieCodec()

	t.Run("creates a snapshot of the (relevant) trie data", func(t *testing.T) {
		trie := newTrie(codec)
		back := newTrie(codec)

		values := []*triehash.TriePair{
			{K: []uint8("test"), V: []uint8("one")},
		}

		root := triehash.TrieRoot(values)

		trie.Put(values[0].K, values[0].V)
		trie.Put(values[0].K, []uint8("two"))
		trie.Del(values[0].K)
		trie.Put(values[0].K, values[0].V)
		trie.Put([]uint8("doge"), []uint8("coin"))
		trie.Del([]uint8("doge"))

		trie.Snapshot(back, nil)

		if !reflect.DeepEqual(back.GetRoot(), trie.GetRoot()) {
			t.Fail()
		}

		if !reflect.DeepEqual(back.GetRoot(), root[:]) {
			t.Fail()
		}

		if !reflect.DeepEqual(trie.Get(values[0].K), values[0].V) {
			t.Fail()
		}
	})

	t.Run("creates a snapshot of the (relevant) data", func(t *testing.T) {
		trie := newTrie(codec)
		back := newTrie(codec)

		// TODO: fix trie encoder to fix tests
		values := []*triehash.TriePair{
			{K: []uint8("one"), V: []uint8("testing")},
			{K: []uint8("two"), V: []uint8("testing with a much longer value here")},
			{K: []uint8("twzei"), V: []uint8("und Deutch")},
			{K: []uint8("do"), V: []uint8("do it")},
			{K: []uint8("dog"), V: []uint8("doggie")},
			&triehash.TriePair{K: []uint8("dogge"), V: []uint8("bigger dog")},
			&triehash.TriePair{K: []uint8("dodge"), V: []uint8("coin")},
		}

		root := triehash.TrieRoot(values)

		for _, value := range values {
			trie.Put(value.K, value.V)
		}

		trie.Snapshot(back, nil)

		if !reflect.DeepEqual(back.GetRoot(), trie.GetRoot()) {
			t.Fail()
		}

		{
			got := hex.EncodeToString(back.GetRoot()[:])
			want := hex.EncodeToString(root[:])
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		}

		{
			got := trie.Get(values[0].K)
			want := values[0].V

			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	})
}
