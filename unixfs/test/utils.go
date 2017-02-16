package testu

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	imp "github.com/ipfs/go-ipfs/importer"
	"github.com/ipfs/go-ipfs/importer/chunk"
	mdag "github.com/ipfs/go-ipfs/merkledag"
	mdagmock "github.com/ipfs/go-ipfs/merkledag/test"
	ft "github.com/ipfs/go-ipfs/unixfs"

	u "gx/ipfs/QmZNVWh8LLjAavuQ2JXuFmuYH3C11xo988vSgp7UQrTRj1/go-ipfs-util"
	context "gx/ipfs/QmZy2y8t9zQH2a1b8q2ZSLKp17ATuJoCNxxyMFG5qFExpt/go-net/context"
)

func SizeSplitterGen(size int64) chunk.SplitterGen {
	return func(r io.Reader) chunk.Splitter {
		return chunk.NewSizeSplitter(r, size)
	}
}

func GetDAGServ() mdag.DAGService {
	return mdagmock.Mock()
}

func GetNode(t testing.TB, dserv mdag.DAGService, data []byte) *mdag.Node {
	in := bytes.NewReader(data)
	node, err := imp.BuildTrickleDagFromReader(dserv, SizeSplitterGen(500)(in))
	if err != nil {
		t.Fatal(err)
	}

	return node
}

func GetEmptyNode(t testing.TB, dserv mdag.DAGService) *mdag.Node {
	return GetNode(t, dserv, []byte{})
}

func GetRandomNode(t testing.TB, dserv mdag.DAGService, size int64) ([]byte, *mdag.Node) {
	in := io.LimitReader(u.NewTimeSeededRand(), size)
	buf, err := ioutil.ReadAll(in)
	if err != nil {
		t.Fatal(err)
	}

	node := GetNode(t, dserv, buf)
	return buf, node
}

func ArrComp(a, b []byte) error {
	if len(a) != len(b) {
		return fmt.Errorf("Arrays differ in length. %d != %d", len(a), len(b))
	}
	for i, v := range a {
		if v != b[i] {
			return fmt.Errorf("Arrays differ at index: %d", i)
		}
	}
	return nil
}

func PrintDag(nd *mdag.Node, ds mdag.DAGService, indent int) {
	pbd, err := ft.FromBytes(nd.Data())
	if err != nil {
		panic(err)
	}

	for i := 0; i < indent; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("{size = %d, type = %s, children = %d", pbd.GetFilesize(), pbd.GetType().String(), len(pbd.GetBlocksizes()))
	if len(nd.Links) > 0 {
		fmt.Println()
	}
	for _, lnk := range nd.Links {
		child, err := lnk.GetNode(context.Background(), ds)
		if err != nil {
			panic(err)
		}
		PrintDag(child, ds, indent+1)
	}
	if len(nd.Links) > 0 {
		for i := 0; i < indent; i++ {
			fmt.Print(" ")
		}
	}
	fmt.Println("}")
}
