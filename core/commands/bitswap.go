package commands

import (
	"bytes"
	"fmt"
	"io"

	cmds "github.com/ipfs/go-ipfs/commands"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	decision "github.com/ipfs/go-ipfs/exchange/bitswap/decision"
	key "gx/ipfs/Qmce4Y4zg3sYr7xKM5UueS67vhNni6EeWgCRnb7MbLJMew/go-key"

	"gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	peer "gx/ipfs/QmWXjJo15p4pzT7cayEwZi2sWgJqLnGDof6ZGMh9xBgU1p/go-libp2p-peer"
	u "gx/ipfs/QmZNVWh8LLjAavuQ2JXuFmuYH3C11xo988vSgp7UQrTRj1/go-ipfs-util"
	cid "gx/ipfs/QmfSc2xehWmWLnwwYR91Y8QF4xdASypTFVknutoKQS3GHp/go-cid"
)

var BitswapCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Interact with the bitswap agent.",
		ShortDescription: ``,
	},
	Subcommands: map[string]*cmds.Command{
		"wantlist": showWantlistCmd,
		"stat":     bitswapStatCmd,
		"unwant":   unwantCmd,
		"ledger":   ledgerCmd,
	},
}

var unwantCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Remove a given block from your wantlist.",
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("key", true, true, "Key(s) to remove from your wantlist.").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(u.ErrCast(), cmds.ErrNormal)
			return
		}

		var ks []key.Key
		for _, arg := range req.Arguments() {
			c, err := cid.Decode(arg)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			ks = append(ks, key.Key(c.Hash()))
		}

		bs.CancelWants(ks)
	},
}

var showWantlistCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Show blocks currently on the wantlist.",
		ShortDescription: `
Print out all blocks currently on the bitswap wantlist for the local peer.`,
	},
	Options: []cmds.Option{
		cmds.StringOption("peer", "p", "Specify which peer to show wantlist for. Default: self."),
	},
	Type: KeyList{},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(u.ErrCast(), cmds.ErrNormal)
			return
		}

		pstr, found, err := req.Option("peer").String()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		if found {
			pid, err := peer.IDB58Decode(pstr)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			res.SetOutput(&KeyList{bs.WantlistForPeer(pid)})
		} else {
			res.SetOutput(&KeyList{bs.GetWantlist()})
		}
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: KeyListTextMarshaler,
	},
}

var bitswapStatCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Show some diagnostic information on the bitswap agent.",
		ShortDescription: ``,
	},
	Type: bitswap.Stat{},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(u.ErrCast(), cmds.ErrNormal)
			return
		}

		st, err := bs.Stat()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(st)
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			out, ok := res.Output().(*bitswap.Stat)
			if !ok {
				return nil, u.ErrCast()
			}
			buf := new(bytes.Buffer)
			fmt.Fprintln(buf, "bitswap status")
			fmt.Fprintf(buf, "\tprovides buffer: %d / %d\n", out.ProvideBufLen, bitswap.HasBlockBufferSize)
			fmt.Fprintf(buf, "\tblocks received: %d\n", out.BlocksReceived)
			fmt.Fprintf(buf, "\tdup blocks received: %d\n", out.DupBlksReceived)
			fmt.Fprintf(buf, "\tdup data received: %s\n", humanize.Bytes(out.DupDataReceived))
			fmt.Fprintf(buf, "\twantlist [%d keys]\n", len(out.Wantlist))
			for _, k := range out.Wantlist {
				fmt.Fprintf(buf, "\t\t%s\n", k.B58String())
			}
			fmt.Fprintf(buf, "\tpartners [%d]\n", len(out.Peers))
			for _, p := range out.Peers {
				fmt.Fprintf(buf, "\t\t%s\n", p)
			}
			return buf, nil
		},
	},
}

var ledgerCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Show the current ledger for a peer.",
		ShortDescription: `
The Bitswap decision engine tracks the number of bytes exchanged between IPFS
nodes, and stores this information as a collection of ledgers. This command
prints the ledger associated with a given peer.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("peer", true, false, "The PeerID (B58) of the ledger to inspect."),
	},
	Type: decision.Receipt{},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(u.ErrCast(), cmds.ErrNormal)
			return
		}

		partner, err := peer.IDB58Decode(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrClient)
			return
		}
		res.SetOutput(bs.LedgerForPeer(partner))
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			out, ok := res.Output().(*decision.Receipt)
			if !ok {
				return nil, u.ErrCast()
			}
			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "Ledger for %s\n"+
				"Debt ratio:\t%f\n"+
				"Exchanges:\t%d\n"+
				"Bytes sent:\t%d\n"+
				"Bytes received:\t%d\n\n",
				out.Peer, out.Value, out.Exchanged,
				out.Sent, out.Recv)
			return buf, nil
		},
	},
}
