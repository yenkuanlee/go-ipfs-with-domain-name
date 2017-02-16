package commands

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	context "gx/ipfs/QmZy2y8t9zQH2a1b8q2ZSLKp17ATuJoCNxxyMFG5qFExpt/go-net/context"

	cmds "github.com/ipfs/go-ipfs/commands"
	core "github.com/ipfs/go-ipfs/core"
	path "github.com/ipfs/go-ipfs/path"
	crypto "gx/ipfs/QmVoi5es8D5fNHZDqoW6DgDAEPEV5hQp8GBz161vZXiwpQ/go-libp2p-crypto"
	key "gx/ipfs/Qmce4Y4zg3sYr7xKM5UueS67vhNni6EeWgCRnb7MbLJMew/go-key"
)

var errNotOnline = errors.New("This command must be run in online mode. Try running 'ipfs daemon' first.")

var PublishCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Publish an object to IPNS.",
		ShortDescription: `
IPNS is a PKI namespace, where names are the hashes of public keys, and
the private key enables publishing new (signed) values. In publish, the
default value of <name> is your own identity public key.
`,
		LongDescription: `
IPNS is a PKI namespace, where names are the hashes of public keys, and
the private key enables publishing new (signed) values. In publish, the
default value of <name> is your own identity public key.

Examples:

Publish an <ipfs-path> to your identity name:

  > ipfs name publish /ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy
  Published to QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n: /ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy

Publish an <ipfs-path> to another public key (not implemented):

  > ipfs name publish /ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n
  Published to QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n: /ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy

`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, false, "IPFS path of the object to be published.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.BoolOption("resolve", "Resolve given path before publishing.").Default(true),
		cmds.StringOption("lifetime", "t",
			`Time duration that the record will be valid for. <<default>>
    This accepts durations such as "300s", "1.5h" or "2h45m". Valid time units are
    "ns", "us" (or "µs"), "ms", "s", "m", "h".`).Default("24h"),
		cmds.StringOption("ttl", "Time duration this record should be cached for (caution: experimental)."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		log.Debug("begin publish")
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if !n.OnlineMode() {
			err := n.SetupOfflineRouting()
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
		}

		if n.Mounts.Ipns != nil && n.Mounts.Ipns.IsActive() {
			res.SetError(errors.New("You cannot manually publish while IPNS is mounted."), cmds.ErrNormal)
			return
		}

		pstr := req.Arguments()[0]

		if n.Identity == "" {
			res.SetError(errors.New("Identity not loaded!"), cmds.ErrNormal)
			return
		}

		popts := new(publishOpts)

		popts.verifyExists, _, _ = req.Option("resolve").Bool()

		validtime, _, _ := req.Option("lifetime").String()
		d, err := time.ParseDuration(validtime)
		if err != nil {
			res.SetError(fmt.Errorf("error parsing lifetime option: %s", err), cmds.ErrNormal)
			return
		}

		popts.pubValidTime = d

		ctx := req.Context()
		if ttl, found, _ := req.Option("ttl").String(); found {
			d, err := time.ParseDuration(ttl)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			ctx = context.WithValue(ctx, "ipns-publish-ttl", d)
		}

		output, err := publish(ctx, n, n.PrivateKey, path.Path(pstr), popts)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		res.SetOutput(output)
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v := res.Output().(*IpnsEntry)
			s := fmt.Sprintf("Published to %s: %s\n", v.Name, v.Value)
			return strings.NewReader(s), nil
		},
	},
	Type: IpnsEntry{},
}

type publishOpts struct {
	verifyExists bool
	pubValidTime time.Duration
}

func publish(ctx context.Context, n *core.IpfsNode, k crypto.PrivKey, ref path.Path, opts *publishOpts) (*IpnsEntry, error) {

	if opts.verifyExists {
		// verify the path exists
		_, err := core.Resolve(ctx, n, ref)
		if err != nil {
			return nil, err
		}
	}

	eol := time.Now().Add(opts.pubValidTime)
	err := n.Namesys.PublishWithEOL(ctx, k, ref, eol)
	if err != nil {
		return nil, err
	}

	hash, err := k.GetPublic().Hash()
	if err != nil {
		return nil, err
	}

	return &IpnsEntry{
		Name:  key.Key(hash).String(),
		Value: ref.String(),
	}, nil
}
