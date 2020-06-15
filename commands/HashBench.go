package commands

import (
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"time"
)

// HashBench benchmarks crc32
type HashBench struct {
	randReader io.Reader
	buff       []byte
}

// NewHashBench create a new hashBench
func NewHashBench() *HashBench {
	return &HashBench{
		randReader: rand.New(rand.NewSource(time.Now().UnixNano())),
		buff:       make([]byte, 1000*1000),
	}
}

// DoTest runs the hash test
func (ht HashBench) DoTest() int {
	var hdv1, hdv2 int
	duration := time.Duration(100 * time.Millisecond)

	// Using 1MB
	var hashSize = int64(1000 * 1000 * 1)

	c := make(chan int, 0)
	quit := make(chan int, 0)

	// Benchmark with 1MB
	ht.benchHash(hashSize, c, quit)

	// Wait duration amount of time
	time.Sleep(duration)

	quit <- 1
	hdv1 = <-c

	// To avoid very weak cpus
	// from getting bured
	if hdv1 > 100 {
		hashSize *= 10
		ht.benchHash(hashSize, c, quit)
		time.Sleep(duration * 10)
		quit <- 1
		hdv2 = <-c

		return ((hdv1 + (hdv2 / 10)) / 2)
	}

	return hdv1
}

func (ht HashBench) benchHash(hashSize int64, c, quit chan int) {
	go func() {
		hashesDone := 0
		for {

			hash := crc32.NewIEEE()
			_, err := io.CopyBuffer(hash, io.LimitReader(ht.randReader, hashSize), ht.buff)

			if err != nil {
				fmt.Println(err)
			}
			hashesDone++

			select {
			case <-quit:
				c <- hashesDone
				return
			default:
			}
		}
	}()
}
