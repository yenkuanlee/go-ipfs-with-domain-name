package commands

import (
	"bytes"
	"fmt"
	"io"

	cmds "github.com/ipfs/go-ipfs/commands"
	core "github.com/ipfs/go-ipfs/core"
	corerepo "github.com/ipfs/go-ipfs/core/corerepo"
	dag "github.com/ipfs/go-ipfs/merkledag"
	path "github.com/ipfs/go-ipfs/path"
	pin "github.com/ipfs/go-ipfs/pin"

	u "gx/ipfs/QmZNVWh8LLjAavuQ2JXuFmuYH3C11xo988vSgp7UQrTRj1/go-ipfs-util"
	context "gx/ipfs/QmZy2y8t9zQH2a1b8q2ZSLKp17ATuJoCNxxyMFG5qFExpt/go-net/context"
	cid "gx/ipfs/QmfSc2xehWmWLnwwYR91Y8QF4xdASypTFVknutoKQS3GHp/go-cid"
)

var PinCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Pin (and unpin) objects to local storage.",
	},

	Subcommands: map[string]*cmds.Command{
		"add": addPinCmd,
		"rm":  rmPinCmd,
		"ls":  listPinCmd,
	},
}

type PinOutput struct {
	Pins []*cid.Cid
}

var addPinCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Pin objects to local storage.",
		ShortDescription: "Stores an IPFS object(s) from a given path locally to disk.",
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, true, "Path to object(s) to be pinned.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.BoolOption("recursive", "r", "Recursively pin the object linked to by the specified object(s).").Default(true),
	},
	Type: PinOutput{},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		defer n.Blockstore.PinLock().Unlock()

		// set recursive flag
		recursive, _, err := req.Option("recursive").Bool()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		added, err := corerepo.Pin(n, req.Context(), req.Arguments(), recursive)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(&PinOutput{added})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			added, ok := res.Output().(*PinOutput)
			if !ok {
				return nil, u.ErrCast()
			}

			var pintype string
			rec, found, _ := res.Request().Option("recursive").Bool()
			if rec || !found {
				pintype = "recursively"
			} else {
				pintype = "directly"
			}

			buf := new(bytes.Buffer)
			for _, k := range added.Pins {
				fmt.Fprintf(buf, "pinned %s %s\n", k, pintype)
			}
			return buf, nil
		},
	},
}

var rmPinCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Remove pinned objects from local storage.",
		ShortDescription: `
Removes the pin from the given object allowing it to be garbage
collected if needed. (By default, recursively. Use -r=false for direct pins.)
`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", true, true, "Path to object(s) to be unpinned.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.BoolOption("recursive", "r", "Recursively unpin the object linked to by the specified object(s).").Default(true),
	},
	Type: PinOutput{},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		// set recursive flag
		recursive, _, err := req.Option("recursive").Bool()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		removed, err := corerepo.Unpin(n, req.Context(), req.Arguments(), recursive)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(&PinOutput{removed})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			added, ok := res.Output().(*PinOutput)
			if !ok {
				return nil, u.ErrCast()
			}

			buf := new(bytes.Buffer)
			for _, k := range added.Pins {
				fmt.Fprintf(buf, "unpinned %s\n", k)
			}
			return buf, nil
		},
	},
}

var listPinCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "List objects pinned to local storage.",
		ShortDescription: `
Returns a list of objects that are pinned locally.
By default, all pinned objects are returned, but the '--type' flag or
arguments can restrict that to a specific pin type or to some specific objects
respectively.
`,
		LongDescription: `
Returns a list of objects that are pinned locally.
By default, all pinned objects are returned, but the '--type' flag or
arguments can restrict that to a specific pin type or to some specific objects
respectively.

Use --type=<type> to specify the type of pinned keys to list.
Valid values are:
    * "direct": pin that specific object.
    * "recursive": pin that specific object, and indirectly pin all its
    	descendants
    * "indirect": pinned indirectly by an ancestor (like a refcount)
    * "all"

With arguments, the command fails if any of the arguments is not a pinned
object. And if --type=<type> is additionally used, the command will also fail
if any of the arguments is not of the specified type.

Example:
	$ echo "hello" | ipfs add -q
	QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN
	$ ipfs pin ls
	QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN recursive
	# now remove the pin, and repin it directly
	$ ipfs pin rm QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN
	unpinned QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN
	$ ipfs pin add -r=false QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN
	pinned QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN directly
	$ ipfs pin ls --type=direct
	QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN direct
	$ ipfs pin ls QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN
	QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN direct
`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("ipfs-path", false, true, "Path to object(s) to be listed."),
	},
	Options: []cmds.Option{
		cmds.StringOption("type", "t", "The type of pinned keys to list. Can be \"direct\", \"indirect\", \"recursive\", or \"all\".").Default("all"),
		cmds.BoolOption("quiet", "q", "Write just hashes of objects.").Default(false),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		typeStr, _, err := req.Option("type").String()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		switch typeStr {
		case "all", "direct", "indirect", "recursive":
		default:
			err = fmt.Errorf("Invalid type '%s', must be one of {direct, indirect, recursive, all}", typeStr)
			res.SetError(err, cmds.ErrClient)
			return
		}

		var keys map[string]RefKeyObject

		if len(req.Arguments()) > 0 {
			keys, err = pinLsKeys(req.Arguments(), typeStr, req.Context(), n)
		} else {
			keys, err = pinLsAll(typeStr, req.Context(), n)
		}

		if err != nil {
			res.SetError(err, cmds.ErrNormal)
		} else {
			res.SetOutput(&RefKeyList{Keys: keys})
		}
	},
	Type: RefKeyList{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			quiet, _, err := res.Request().Option("quiet").Bool()
			if err != nil {
				return nil, err
			}

			keys, ok := res.Output().(*RefKeyList)
			if !ok {
				return nil, u.ErrCast()
			}
			out := new(bytes.Buffer)
			for k, v := range keys.Keys {
				if quiet {
					fmt.Fprintf(out, "%s\n", k)
				} else {
					fmt.Fprintf(out, "%s %s\n", k, v.Type)
				}
			}
			return out, nil
		},
	},
}

type RefKeyObject struct {
	Type string
}

type RefKeyList struct {
	Keys map[string]RefKeyObject
}

func pinLsKeys(args []string, typeStr string, ctx context.Context, n *core.IpfsNode) (map[string]RefKeyObject, error) {

	keys := make(map[string]RefKeyObject)

	for _, p := range args {
		pth, err := path.ParsePath(p)
		if err != nil {
			return nil, err
		}

		dagNode, err := core.Resolve(ctx, n, pth)
		if err != nil {
			return nil, err
		}

		mode, ok := pin.StringToPinMode(typeStr)
		if !ok {
			return nil, fmt.Errorf("Invalid pin mode '%s'", typeStr)
		}

		c := dagNode.Cid()
		pinType, pinned, err := n.Pinning.IsPinnedWithType(c, mode)
		if err != nil {
			return nil, err
		}

		if !pinned {
			return nil, fmt.Errorf("Path '%s' is not pinned", p)
		}

		switch pinType {
		case "direct", "indirect", "recursive", "internal":
		default:
			pinType = "indirect through " + pinType
		}
		keys[c.String()] = RefKeyObject{
			Type: pinType,
		}
	}

	return keys, nil
}

func pinLsAll(typeStr string, ctx context.Context, n *core.IpfsNode) (map[string]RefKeyObject, error) {

	keys := make(map[string]RefKeyObject)

	AddToResultKeys := func(keyList []*cid.Cid, typeStr string) {
		for _, c := range keyList {
			keys[c.String()] = RefKeyObject{
				Type: typeStr,
			}
		}
	}

	if typeStr == "direct" || typeStr == "all" {
		AddToResultKeys(n.Pinning.DirectKeys(), "direct")
	}
	if typeStr == "indirect" || typeStr == "all" {
		set := cid.NewSet()
		for _, k := range n.Pinning.RecursiveKeys() {
			nd, err := n.DAG.Get(ctx, k)
			if err != nil {
				return nil, err
			}

			err = dag.EnumerateChildren(n.Context(), n.DAG, nd, set.Visit, false)
			if err != nil {
				return nil, err
			}
		}
		AddToResultKeys(set.Keys(), "indirect")
	}
	if typeStr == "recursive" || typeStr == "all" {
		AddToResultKeys(n.Pinning.RecursiveKeys(), "recursive")
	}

	return keys, nil
}
