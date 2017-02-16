package corehttp

import (
	"net"
	"net/http"
	"strings"

	"github.com/ipfs/go-ipfs/core"
	"gx/ipfs/QmZy2y8t9zQH2a1b8q2ZSLKp17ATuJoCNxxyMFG5qFExpt/go-net/context"
	isd "gx/ipfs/QmaeHSCBd9XjXxmgHEiKkHtLcMCb2eZsPLKT7bHgBfBkqw/go-is-domain"
)

// IPNSHostnameOption rewrites an incoming request if its Host: header contains
// an IPNS name.
// The rewritten request points at the resolved name on the gateway handler.
func IPNSHostnameOption() ServeOption {
	return func(n *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		childMux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithCancel(n.Context())
			defer cancel()

			host := strings.SplitN(r.Host, ":", 2)[0]
			if len(host) > 0 && isd.IsDomain(host) {
				name := "/ipns/" + host
				if _, err := n.Namesys.Resolve(ctx, name); err == nil {
					r.Header["X-Ipns-Original-Path"] = []string{r.URL.Path}
					r.URL.Path = name + r.URL.Path
				}
			}
			childMux.ServeHTTP(w, r)
		})
		return childMux, nil
	}
}
