package bitswap

import (
	"sync"
	"time"

	engine "github.com/ipfs/go-ipfs/exchange/bitswap/decision"
	bsmsg "github.com/ipfs/go-ipfs/exchange/bitswap/message"
	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	wantlist "github.com/ipfs/go-ipfs/exchange/bitswap/wantlist"
	peer "gx/ipfs/QmWXjJo15p4pzT7cayEwZi2sWgJqLnGDof6ZGMh9xBgU1p/go-libp2p-peer"
	context "gx/ipfs/QmZy2y8t9zQH2a1b8q2ZSLKp17ATuJoCNxxyMFG5qFExpt/go-net/context"
	key "gx/ipfs/Qmce4Y4zg3sYr7xKM5UueS67vhNni6EeWgCRnb7MbLJMew/go-key"
)

type WantManager struct {
	// sync channels for Run loop
	incoming   chan []*bsmsg.Entry
	connect    chan peer.ID        // notification channel for new peers connecting
	disconnect chan peer.ID        // notification channel for peers disconnecting
	peerReqs   chan chan []peer.ID // channel to request connected peers on

	// synchronized by Run loop, only touch inside there
	peers map[peer.ID]*msgQueue
	wl    *wantlist.ThreadSafe

	network bsnet.BitSwapNetwork
	ctx     context.Context
	cancel  func()
}

func NewWantManager(ctx context.Context, network bsnet.BitSwapNetwork) *WantManager {
	ctx, cancel := context.WithCancel(ctx)
	return &WantManager{
		incoming:   make(chan []*bsmsg.Entry, 10),
		connect:    make(chan peer.ID, 10),
		disconnect: make(chan peer.ID, 10),
		peerReqs:   make(chan chan []peer.ID),
		peers:      make(map[peer.ID]*msgQueue),
		wl:         wantlist.NewThreadSafe(),
		network:    network,
		ctx:        ctx,
		cancel:     cancel,
	}
}

type msgPair struct {
	to  peer.ID
	msg bsmsg.BitSwapMessage
}

type cancellation struct {
	who peer.ID
	blk key.Key
}

type msgQueue struct {
	p peer.ID

	outlk   sync.Mutex
	out     bsmsg.BitSwapMessage
	network bsnet.BitSwapNetwork

	sender bsnet.MessageSender

	refcnt int

	work chan struct{}
	done chan struct{}
}

func (pm *WantManager) WantBlocks(ctx context.Context, ks []key.Key) {
	log.Infof("want blocks: %s", ks)
	pm.addEntries(ctx, ks, false)
}

func (pm *WantManager) CancelWants(ks []key.Key) {
	log.Infof("cancel wants: %s", ks)
	pm.addEntries(context.TODO(), ks, true)
}

func (pm *WantManager) addEntries(ctx context.Context, ks []key.Key, cancel bool) {
	var entries []*bsmsg.Entry
	for i, k := range ks {
		entries = append(entries, &bsmsg.Entry{
			Cancel: cancel,
			Entry: &wantlist.Entry{
				Key:      k,
				Priority: kMaxPriority - i,
				RefCnt:   1,
			},
		})
	}
	select {
	case pm.incoming <- entries:
	case <-pm.ctx.Done():
	case <-ctx.Done():
	}
}

func (pm *WantManager) ConnectedPeers() []peer.ID {
	resp := make(chan []peer.ID)
	pm.peerReqs <- resp
	return <-resp
}

func (pm *WantManager) SendBlock(ctx context.Context, env *engine.Envelope) {
	// Blocks need to be sent synchronously to maintain proper backpressure
	// throughout the network stack
	defer env.Sent()

	msg := bsmsg.New(false)
	msg.AddBlock(env.Block)
	log.Infof("Sending block %s to %s", env.Block, env.Peer)
	err := pm.network.SendMessage(ctx, env.Peer, msg)
	if err != nil {
		log.Infof("sendblock error: %s", err)
	}
}

func (pm *WantManager) startPeerHandler(p peer.ID) *msgQueue {
	mq, ok := pm.peers[p]
	if ok {
		mq.refcnt++
		return nil
	}

	mq = pm.newMsgQueue(p)

	// new peer, we will want to give them our full wantlist
	fullwantlist := bsmsg.New(true)
	for _, e := range pm.wl.Entries() {
		fullwantlist.AddEntry(e.Key, e.Priority)
	}
	mq.out = fullwantlist
	mq.work <- struct{}{}

	pm.peers[p] = mq
	go mq.runQueue(pm.ctx)
	return mq
}

func (pm *WantManager) stopPeerHandler(p peer.ID) {
	pq, ok := pm.peers[p]
	if !ok {
		// TODO: log error?
		return
	}

	pq.refcnt--
	if pq.refcnt > 0 {
		return
	}

	close(pq.done)
	delete(pm.peers, p)
}

func (mq *msgQueue) runQueue(ctx context.Context) {
	defer func() {
		if mq.sender != nil {
			mq.sender.Close()
		}
	}()
	for {
		select {
		case <-mq.work: // there is work to be done
			mq.doWork(ctx)
		case <-mq.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (mq *msgQueue) doWork(ctx context.Context) {
	// allow ten minutes for connections
	// this includes looking them up in the dht
	// dialing them, and handshaking
	if mq.sender == nil {
		conctx, cancel := context.WithTimeout(ctx, time.Minute*10)
		defer cancel()

		err := mq.network.ConnectTo(conctx, mq.p)
		if err != nil {
			log.Infof("cant connect to peer %s: %s", mq.p, err)
			// TODO: cant connect, what now?
			return
		}

		nsender, err := mq.network.NewMessageSender(ctx, mq.p)
		if err != nil {
			log.Infof("cant open new stream to peer %s: %s", mq.p, err)
			// TODO: cant open stream, what now?
			return
		}

		mq.sender = nsender
	}

	// grab outgoing message
	mq.outlk.Lock()
	wlm := mq.out
	if wlm == nil || wlm.Empty() {
		mq.outlk.Unlock()
		return
	}
	mq.out = nil
	mq.outlk.Unlock()

	// send wantlist updates
	err := mq.sender.SendMsg(wlm)
	if err != nil {
		log.Infof("bitswap send error: %s", err)
		mq.sender.Close()
		mq.sender = nil
		// TODO: what do we do if this fails?
		return
	}
}

func (pm *WantManager) Connected(p peer.ID) {
	select {
	case pm.connect <- p:
	case <-pm.ctx.Done():
	}
}

func (pm *WantManager) Disconnected(p peer.ID) {
	select {
	case pm.disconnect <- p:
	case <-pm.ctx.Done():
	}
}

// TODO: use goprocess here once i trust it
func (pm *WantManager) Run() {
	tock := time.NewTicker(rebroadcastDelay.Get())
	defer tock.Stop()
	for {
		select {
		case entries := <-pm.incoming:

			// add changes to our wantlist
			var filtered []*bsmsg.Entry
			for _, e := range entries {
				if e.Cancel {
					if pm.wl.Remove(e.Key) {
						filtered = append(filtered, e)
					}
				} else {
					if pm.wl.AddEntry(e.Entry) {
						filtered = append(filtered, e)
					}
				}
			}

			// broadcast those wantlist changes
			for _, p := range pm.peers {
				p.addMessage(filtered)
			}

		case <-tock.C:
			// resend entire wantlist every so often (REALLY SHOULDNT BE NECESSARY)
			var es []*bsmsg.Entry
			for _, e := range pm.wl.Entries() {
				es = append(es, &bsmsg.Entry{Entry: e})
			}

			for _, p := range pm.peers {
				p.outlk.Lock()
				p.out = bsmsg.New(true)
				p.outlk.Unlock()

				p.addMessage(es)
			}
		case p := <-pm.connect:
			pm.startPeerHandler(p)
		case p := <-pm.disconnect:
			pm.stopPeerHandler(p)
		case req := <-pm.peerReqs:
			var peers []peer.ID
			for p := range pm.peers {
				peers = append(peers, p)
			}
			req <- peers
		case <-pm.ctx.Done():
			return
		}
	}
}

func (wm *WantManager) newMsgQueue(p peer.ID) *msgQueue {
	mq := new(msgQueue)
	mq.done = make(chan struct{})
	mq.work = make(chan struct{}, 1)
	mq.network = wm.network
	mq.p = p
	mq.refcnt = 1

	return mq
}

func (mq *msgQueue) addMessage(entries []*bsmsg.Entry) {
	mq.outlk.Lock()
	defer func() {
		mq.outlk.Unlock()
		select {
		case mq.work <- struct{}{}:
		default:
		}
	}()

	// if we have no message held, or the one we are given is full
	// overwrite the one we are holding
	if mq.out == nil {
		mq.out = bsmsg.New(false)
	}

	// TODO: add a msg.Combine(...) method
	// otherwise, combine the one we are holding with the
	// one passed in
	for _, e := range entries {
		if e.Cancel {
			mq.out.Cancel(e.Key)
		} else {
			mq.out.AddEntry(e.Key, e.Priority)
		}
	}
}
