package blockstore

import (
	key "gx/ipfs/Qmce4Y4zg3sYr7xKM5UueS67vhNni6EeWgCRnb7MbLJMew/go-key"

	"github.com/ipfs/go-ipfs/blocks"

	"gx/ipfs/QmRg1gKTHzc3CZXSKzem8aR4E3TubFhbgXwfVuWnSK5CC5/go-metrics-interface"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	context "gx/ipfs/QmZy2y8t9zQH2a1b8q2ZSLKp17ATuJoCNxxyMFG5qFExpt/go-net/context"
	ds "gx/ipfs/QmbzuUusHqaLLoNTDEVLcSF6vZDHZDLPC7p4bztRvvkXxU/go-datastore"
)

type arccache struct {
	arc        *lru.ARCCache
	blockstore Blockstore

	hits  metrics.Counter
	total metrics.Counter
}

func newARCCachedBS(ctx context.Context, bs Blockstore, lruSize int) (*arccache, error) {
	arc, err := lru.NewARC(lruSize)
	if err != nil {
		return nil, err
	}
	c := &arccache{arc: arc, blockstore: bs}
	c.hits = metrics.NewCtx(ctx, "arc.hits_total", "Number of ARC cache hits").Counter()
	c.total = metrics.NewCtx(ctx, "arc_total", "Total number of ARC cache requests").Counter()

	return c, nil
}

func (b *arccache) DeleteBlock(k key.Key) error {
	if has, ok := b.hasCached(k); ok && !has {
		return ErrNotFound
	}

	b.arc.Remove(k) // Invalidate cache before deleting.
	err := b.blockstore.DeleteBlock(k)
	switch err {
	case nil, ds.ErrNotFound, ErrNotFound:
		b.arc.Add(k, false)
		return err
	default:
		return err
	}
}

// if ok == false has is inconclusive
// if ok == true then has respons to question: is it contained
func (b *arccache) hasCached(k key.Key) (has bool, ok bool) {
	b.total.Inc()
	if k == "" {
		// Return cache invalid so the call to blockstore happens
		// in case of invalid key and correct error is created.
		return false, false
	}

	h, ok := b.arc.Get(k)
	if ok {
		b.hits.Inc()
		return h.(bool), true
	}
	return false, false
}

func (b *arccache) Has(k key.Key) (bool, error) {
	if has, ok := b.hasCached(k); ok {
		return has, nil
	}

	res, err := b.blockstore.Has(k)
	if err == nil {
		b.arc.Add(k, res)
	}
	return res, err
}

func (b *arccache) Get(k key.Key) (blocks.Block, error) {
	if has, ok := b.hasCached(k); ok && !has {
		return nil, ErrNotFound
	}

	bl, err := b.blockstore.Get(k)
	if bl == nil && err == ErrNotFound {
		b.arc.Add(k, false)
	} else if bl != nil {
		b.arc.Add(k, true)
	}
	return bl, err
}

func (b *arccache) Put(bl blocks.Block) error {
	if has, ok := b.hasCached(bl.Key()); ok && has {
		return nil
	}

	err := b.blockstore.Put(bl)
	if err == nil {
		b.arc.Add(bl.Key(), true)
	}
	return err
}

func (b *arccache) PutMany(bs []blocks.Block) error {
	var good []blocks.Block
	for _, block := range bs {
		// call put on block if result is inconclusive or we are sure that
		// the block isn't in storage
		if has, ok := b.hasCached(block.Key()); !ok || (ok && !has) {
			good = append(good, block)
		}
	}
	err := b.blockstore.PutMany(good)
	if err != nil {
		return err
	}
	for _, block := range good {
		b.arc.Add(block.Key(), true)
	}
	return nil
}

func (b *arccache) AllKeysChan(ctx context.Context) (<-chan key.Key, error) {
	return b.blockstore.AllKeysChan(ctx)
}

func (b *arccache) GCLock() Unlocker {
	return b.blockstore.(GCBlockstore).GCLock()
}

func (b *arccache) PinLock() Unlocker {
	return b.blockstore.(GCBlockstore).PinLock()
}

func (b *arccache) GCRequested() bool {
	return b.blockstore.(GCBlockstore).GCRequested()
}
