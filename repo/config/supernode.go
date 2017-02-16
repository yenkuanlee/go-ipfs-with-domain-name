package config

import "github.com/ipfs/go-ipfs/thirdparty/ipfsaddr"

// TODO replace with final servers before merge

// TODO rename
type SupernodeClientConfig struct {
	Servers []string
}

var DefaultSNRServers = []string{
}

func initSNRConfig() (*SupernodeClientConfig, error) {
	// TODO perform validation
	return &SupernodeClientConfig{
		Servers: DefaultSNRServers,
	}, nil
}

func (gcr *SupernodeClientConfig) ServerIPFSAddrs() ([]ipfsaddr.IPFSAddr, error) {
	var addrs []ipfsaddr.IPFSAddr
	for _, server := range gcr.Servers {
		addr, err := ipfsaddr.ParseString(server)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
