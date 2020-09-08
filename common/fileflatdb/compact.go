package fileflatdb

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"os"
	"syscall"
	"time"

	"github.com/tsfdsong/go-polkadot/common/db"
)

// Compact ...
type Compact struct {
	fd   int64
	file string
}

// NewCompact ...
func NewCompact(file string) *Compact {
	return &Compact{
		fd:   -1,
		file: file,
	}
}

// Maintain ...
func (c *Compact) Maintain(fn *db.ProgressCB) {
	if c.fd != -1 {
		log.Fatalln("[fileflatdb/compact] database cannot be open for compacting")
	}

	start := time.Now().Unix()
	newFile := fmt.Sprintf("%s.compacted", c.file)
	newFd := c.Open(newFile, true)
	oldFd := c.Open(c.file, false)
	keys := c.Compact(*fn, newFd, oldFd)

	closeFd(oldFd)
	closeFd(newFd)

	newStat, err := os.Stat(newFile)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to stat new file: %s\n", err)
	}
	oldStat, err := os.Stat(c.file)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to stat old file: %s\n", err)
	}
	percentage := 100 * (newStat.Size() / oldStat.Size())
	sizeMB := newStat.Size() / (1024 * 1024)
	elapsed := time.Now().Unix() - start

	err = os.Remove(c.file)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to remove file: %s\n", err)
	}
	err = os.Rename(newFile, c.file)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to rename file: %s\n", err)
	}

	log.Printf("compacted in %d, %dk keys, %dMB (%d%%)", elapsed, keys/1e3, sizeMB, percentage)
}

// Open ...
func (c *Compact) Open(file string, startEmpty bool) uintptr {
	_, err := os.Stat(file)
	isExisting := !os.IsNotExist(err)
	if !isExisting || startEmpty {
		data := make([]byte, branchSize)
		err := ioutil.WriteFile(file, data, os.ModePerm)
		if err != nil {
			log.Fatalf("[fileflatdb/compact] failed write to file after opening: %s\n", err)
		}
	}

	f, err := os.OpenFile(file, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed write to open file: %s\n", err)
	}

	return f.Fd()
}

// doCompact ...
func (c *Compact) doCompact(keys *int, percent int, fn db.ProgressCB, newFd, oldFd uintptr, newAt int, oldAt int, depth int) {
	increment := (100 / float64(entryNum)) / math.Pow(float64(entryNum), float64(depth))

	for index := 0; index < entryNum; index++ {
		entry := c.CompactReadEntry(oldFd, oldAt, index)
		dataAt := new(big.Int)
		dataAt.SetBytes(entry[1 : 1+uintSize])
		entryType := entry[0]

		if int(entryType) == SlotEmpty {
			percent += int(increment)
		} else if int(entryType) == SlotLeaf {
			key, value := c.CompactReadKey(oldFd, int64(dataAt.Uint64()))
			keyAt := c.CompactWriteKey(newFd, key, value)

			c.CompactUpdateLink(newFd, newAt, index, keyAt, SlotLeaf)

			newKeys := *keys + 1
			keys = &newKeys
			percent += int(increment)
		} else if int(entryType) == SlotBranch {
			headerAt := c.CompactWriteHeader(newFd, newAt, index)

			c.doCompact(keys, percent, fn, newFd, oldFd, int(headerAt), int(dataAt.Uint64()), depth+1)
		} else {
			log.Fatalf("[fileflatdb/compact] unknown entry type %d\n", entryType)
		}

		var isCompleted bool
		if depth == 0 && index == entryNum-1 {
			isCompleted = true
		}

		if fn != nil {
			fn(&db.ProgressValue{
				IsCompleted: isCompleted,
				Keys:        *keys,
				Percent:     percent,
			})
		}
	}
}

// Compact ...
func (c *Compact) Compact(fn db.ProgressCB, newFd, oldFd uintptr) int {
	var keys int
	var percent int

	c.doCompact(&keys, percent, fn, newFd, oldFd, 0, 0, 0)

	return keys
}

// CompactReadEntry ...
func (c *Compact) CompactReadEntry(fd uintptr, at int, index int) []byte {
	entry := make([]byte, entrySize)
	entryAt := at + (index * entrySize)

	file := os.NewFile(fd, "temp")
	_, err := file.ReadAt(entry, int64(entryAt))
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to read entry: %s\n", err)
	}

	return entry
}

// CompactReadKey ...
func (c *Compact) CompactReadKey(fd uintptr, at int64) ([]byte, []byte) {
	key := make([]byte, keyTotalSize)
	file := os.NewFile(fd, "temp")
	_, err := file.ReadAt(key, at)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to read key: %s\n", err)
	}

	valueLength := new(big.Int)
	valueLength.SetBytes(key[keySize : keySize+uintSize])

	valueAt := new(big.Int)
	valueAt.SetBytes(key[(keySize + uintSize) : (keySize+uintSize)+uintSize])

	value := make([]byte, valueLength.Uint64())
	_, err = file.ReadAt(value, int64(valueAt.Uint64()))
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to read value: %s\n", err)
	}

	return key, value
}

// CompactWriteKey ...
func (c *Compact) CompactWriteKey(fd uintptr, key, value []byte) int64 {
	file := os.NewFile(fd, "temp")
	stat, err := file.Stat()
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to stat file: %s\n", err)
	}
	valueAt := stat.Size()
	keyAt := valueAt + int64(len(value))

	writeUIntBE(key, int64(valueAt), int64(keySize)+int64(uintSize), int64(uintSize))

	_, err = file.WriteAt(value, valueAt)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to write value: %s\n", err)
	}
	_, err = file.WriteAt(key, keyAt)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to compact write key: %s\n", err)
	}

	return keyAt
}

// CompactUpdateLink ...
func (c *Compact) CompactUpdateLink(fd uintptr, at int, index int, pointer int64, kind int) {
	entry := make([]byte, entrySize)
	entryAt := at + (index * entrySize)

	entry[0] = byte(kind)
	writeUIntBE(entry, int64(pointer), int64(1), int64(uintSize))

	file := os.NewFile(fd, "temp")
	_, err := file.WriteAt(entry, int64(entryAt))
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to compact update link write entry: %s\n", err)
	}
}

// CompactWriteHeader ...
func (c *Compact) CompactWriteHeader(fd uintptr, at int, index int) int64 {
	file := os.NewFile(fd, "temp")
	stat, err := file.Stat()
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to stat file: %s\n", err)
	}
	headerAt := stat.Size()

	header := make([]byte, branchSize)
	_, err = file.WriteAt(header, headerAt)
	if err != nil {
		log.Fatalf("[fileflatdb/compact] failed to write header: %s\n", err)
	}

	c.CompactUpdateLink(fd, at, index, headerAt, SlotBranch)

	return headerAt
}

func closeFd(fd uintptr) {
	// close file descriptor
	if err := syscall.Close(int(fd)); err != nil {
		log.Fatalf("[fileflatdb/compact] failed to close file descriptor: %s\n", err)
	}
}
