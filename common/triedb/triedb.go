package triedb

import (
	"fmt"
	"log"
	"time"

	"github.com/tsfdsong/go-polkadot/common/db"
	"github.com/tsfdsong/go-polkadot/common/triecodec"
	"github.com/tsfdsong/go-polkadot/common/triehash"
)

// TrieDB ...
type TrieDB struct {
	impl  *Impl
	Debug bool
}

// NewTrieDB ...
func NewTrieDB(db db.TXDB, rootHash []byte, codec InterfaceCodec) *TrieDB {
	impl := NewImpl(db, rootHash, codec)
	return &TrieDB{
		impl:  impl,
		Debug: false,
	}
}

// SetDebug ...
func (t *TrieDB) SetDebug(enabled bool) {
	t.Debug = enabled
	t.impl.Debug = enabled
}

// DebugLog ...
func (t *TrieDB) DebugLog(i ...interface{}) {
	if t.Debug {
		i = append([]interface{}{"Debug: triedb"}, i...)
		fmt.Println(i...)
	}
}

// Transaction ...
func (t *TrieDB) Transaction(fn func() bool) (bool, error) {
	t.impl.checkpoint.CreateCheckpoint()

	result, err := t.impl.db.Transaction(fn)
	if err != nil {
		t.impl.checkpoint.RevertCheckpoint()
		return false, nil
	}

	if result {
		t.impl.checkpoint.CommitCheckpoint()
	} else {
		t.impl.checkpoint.RevertCheckpoint()
	}

	return result, nil
}

// Open ...
func (t *TrieDB) Open() {
	t.impl.db.Open()
}

// Close ...
func (t *TrieDB) Close() {
	t.impl.db.Close()
}

// Empty ...
func (t *TrieDB) Empty() {
	t.impl.db.Empty()
}

// Drop ...
func (t *TrieDB) Drop() {
	t.impl.db.Drop()
}

// Maintain ...
func (t *TrieDB) Maintain(fn *db.ProgressCB) error {
	t.impl.db.Maintain(fn)
	return nil
}

// Rename ...
func (t *TrieDB) Rename(base, file string) {
	t.impl.db.Rename(base, file)
}

// Size ...
func (t *TrieDB) Size() int {
	return t.impl.db.Size()
}

// Del ...
func (t *TrieDB) Del(key []uint8) {
	t.DebugLog("Del, root hash", t.impl.checkpoint.rootHash)
	n := t.impl.GetNode(t.impl.checkpoint.rootHash)
	t.DebugLog("Del, get node", n)
	nibbles := triecodec.ToNibbles(key)
	t.DebugLog("Del, nibbles key", nibbles)
	node := t.impl.Del(
		n,
		nibbles,
	)

	t.DebugLog("trie Del set root node, node", node)

	t.impl.SetRootNode(node)
}

// Get ...
func (t *TrieDB) Get(key []uint8) []uint8 {
	t.DebugLog("Get, key str", string(key))
	t.DebugLog("Get, root hash", t.impl.checkpoint.rootHash)
	x := t.impl.GetNode(t.impl.checkpoint.rootHash)
	t.DebugLog("Get, node", x)
	nibs := triecodec.ToNibbles(key)
	t.DebugLog("Get, nibbles", nibs)
	value := t.impl.Get(
		x,
		nibs,
	)

	if value == nil {
		t.DebugLog("Get, value is nil")
		return nil
	}

	return value.([]uint8)
}

// Put ...
func (t *TrieDB) Put(key, value []uint8) {
	t.DebugLog("Put, key str", string(key))
	t.DebugLog("Put, value", string(value))
	n := t.impl.GetNode(t.impl.checkpoint.rootHash)
	t.DebugLog("Put, get node", n)
	nibs := triecodec.ToNibbles(key)
	t.DebugLog("Put, nibbles", nibs)
	t.DebugLog("Put, value", value)
	node := t.impl.Put(
		n,
		nibs,
		value,
	)
	t.DebugLog("Put, receive node", node)

	t.impl.SetRootNode(node)
}

// GetRoot ...
func (t *TrieDB) GetRoot() []byte {
	t.DebugLog("get root")
	rootnode := t.GetNode(nil)
	t.DebugLog("get root, root node", rootnode)

	if IsNull(rootnode) {
		t.DebugLog("get root, root node is nil")
		return triehash.TrieRoot(nil)[:]
	}

	return t.impl.checkpoint.rootHash[:]
}

// GetNode ...
func (t *TrieDB) GetNode(hash []byte) Node {
	t.DebugLog("get node, input hash", hash)
	if hash == nil {
		hash = t.impl.checkpoint.rootHash
	}
	t.DebugLog("get node, hash", hash)

	return t.impl.GetNode(hash)
}

// SetRoot ...
func (t *TrieDB) SetRoot(rootHash []byte) {
	t.DebugLog("set root, root hash", rootHash)
	t.impl.checkpoint.rootHash = rootHash
}

// Snapshot ...
func (t *TrieDB) Snapshot(dest *TrieDB, fn db.ProgressCB) int {
	start := time.Now().Unix()

	keys := t.impl.Snapshot(dest, fn, t.impl.checkpoint.rootHash, 0, 0, 0)
	elapsed := time.Now().Unix() - start

	dest.SetRoot(t.impl.checkpoint.rootHash)

	newSize := dest.impl.db.Size()
	t.DebugLog("Snapshot, new size", newSize)
	currentSize := t.impl.db.Size()
	t.DebugLog("Snapshot, current size", currentSize)
	percentage := 100 * (newSize / currentSize)
	sizeMB := newSize / (1024 * 1024)

	log.Printf("snapshot created in %d, %dk keys, %dMB (%d%%)", elapsed, keys/1e3, sizeMB, percentage)

	if fn != nil {
		fn(&db.ProgressValue{
			IsCompleted: true,
			Keys:        keys,
			Percent:     100,
		})
	}

	return keys
}
