// package wantlist implements an object for bitswap that contains the keys
// that a given peer wants.
package wantlist

import (
	"sort"
	"sync"

	key "gx/ipfs/Qmce4Y4zg3sYr7xKM5UueS67vhNni6EeWgCRnb7MbLJMew/go-key"
)

type ThreadSafe struct {
	lk       sync.RWMutex
	Wantlist Wantlist
}

// not threadsafe
type Wantlist struct {
	set map[key.Key]*Entry
}

type Entry struct {
	Key      key.Key
	Priority int

	RefCnt int
}

type entrySlice []*Entry

func (es entrySlice) Len() int           { return len(es) }
func (es entrySlice) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es entrySlice) Less(i, j int) bool { return es[i].Priority > es[j].Priority }

func NewThreadSafe() *ThreadSafe {
	return &ThreadSafe{
		Wantlist: *New(),
	}
}

func New() *Wantlist {
	return &Wantlist{
		set: make(map[key.Key]*Entry),
	}
}

func (w *ThreadSafe) Add(k key.Key, priority int) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	return w.Wantlist.Add(k, priority)
}

func (w *ThreadSafe) AddEntry(e *Entry) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	return w.Wantlist.AddEntry(e)
}

func (w *ThreadSafe) Remove(k key.Key) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	return w.Wantlist.Remove(k)
}

func (w *ThreadSafe) Contains(k key.Key) (*Entry, bool) {
	w.lk.RLock()
	defer w.lk.RUnlock()
	return w.Wantlist.Contains(k)
}

func (w *ThreadSafe) Entries() []*Entry {
	w.lk.RLock()
	defer w.lk.RUnlock()
	return w.Wantlist.Entries()
}

func (w *ThreadSafe) SortedEntries() []*Entry {
	w.lk.RLock()
	defer w.lk.RUnlock()
	return w.Wantlist.SortedEntries()
}

func (w *ThreadSafe) Len() int {
	w.lk.RLock()
	defer w.lk.RUnlock()
	return w.Wantlist.Len()
}

func (w *Wantlist) Len() int {
	return len(w.set)
}

func (w *Wantlist) Add(k key.Key, priority int) bool {
	if e, ok := w.set[k]; ok {
		e.RefCnt++
		return false
	}

	w.set[k] = &Entry{
		Key:      k,
		Priority: priority,
		RefCnt:   1,
	}

	return true
}

func (w *Wantlist) AddEntry(e *Entry) bool {
	if ex, ok := w.set[e.Key]; ok {
		ex.RefCnt++
		return false
	}
	w.set[e.Key] = e
	return true
}

func (w *Wantlist) Remove(k key.Key) bool {
	e, ok := w.set[k]
	if !ok {
		return false
	}

	e.RefCnt--
	if e.RefCnt <= 0 {
		delete(w.set, k)
		return true
	}
	return false
}

func (w *Wantlist) Contains(k key.Key) (*Entry, bool) {
	e, ok := w.set[k]
	return e, ok
}

func (w *Wantlist) Entries() []*Entry {
	var es entrySlice
	for _, e := range w.set {
		es = append(es, e)
	}
	return es
}

func (w *Wantlist) SortedEntries() []*Entry {
	var es entrySlice
	for _, e := range w.set {
		es = append(es, e)
	}
	sort.Sort(es)
	return es
}
