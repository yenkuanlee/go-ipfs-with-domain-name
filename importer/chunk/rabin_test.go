package chunk

import (
	"bytes"
	"fmt"
	"github.com/ipfs/go-ipfs/blocks"
	"gx/ipfs/QmZNVWh8LLjAavuQ2JXuFmuYH3C11xo988vSgp7UQrTRj1/go-ipfs-util"
	"gx/ipfs/Qmce4Y4zg3sYr7xKM5UueS67vhNni6EeWgCRnb7MbLJMew/go-key"
	"io"
	"testing"
)

func TestRabinChunking(t *testing.T) {
	data := make([]byte, 1024*1024*16)
	util.NewTimeSeededRand().Read(data)

	r := NewRabin(bytes.NewReader(data), 1024*256)

	var chunks [][]byte

	for {
		chunk, err := r.NextBytes()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}

		chunks = append(chunks, chunk)
	}

	fmt.Printf("average block size: %d\n", len(data)/len(chunks))

	unchunked := bytes.Join(chunks, nil)
	if !bytes.Equal(unchunked, data) {
		fmt.Printf("%d %d\n", len(unchunked), len(data))
		t.Fatal("data was chunked incorrectly")
	}
}

func chunkData(t *testing.T, data []byte) map[key.Key]blocks.Block {
	r := NewRabin(bytes.NewReader(data), 1024*256)

	blkmap := make(map[key.Key]blocks.Block)

	for {
		blk, err := r.NextBytes()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}

		b := blocks.NewBlock(blk)
		blkmap[b.Key()] = b
	}

	return blkmap
}

func TestRabinChunkReuse(t *testing.T) {
	data := make([]byte, 1024*1024*16)
	util.NewTimeSeededRand().Read(data)

	ch1 := chunkData(t, data[1000:])
	ch2 := chunkData(t, data)

	var extra int
	for k, _ := range ch2 {
		_, ok := ch1[k]
		if !ok {
			extra++
		}
	}

	if extra > 2 {
		t.Log("too many spare chunks made")
	}
}
