package config

import (
"os/exec"
"bufio"
"os"
"strings"

 "io/ioutil"

	"encoding/base64"
	"errors"
	"fmt"
	"io"

	ci "gx/ipfs/QmVoi5es8D5fNHZDqoW6DgDAEPEV5hQp8GBz161vZXiwpQ/go-libp2p-crypto"
	peer "gx/ipfs/QmWXjJo15p4pzT7cayEwZi2sWgJqLnGDof6ZGMh9xBgU1p/go-libp2p-peer"
)

func Init(out io.Writer, nBitsForKeypair int) (*Config, error) {
	identity, err := identityConfig(out, nBitsForKeypair)
	if err != nil {
		return nil, err
	}

	bootstrapPeers, err := DefaultBootstrapPeers()
	if err != nil {
		return nil, err
	}

	datastore, err := datastoreConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{

		// setup the node's default addresses.
		// NOTE: two swarm listen addrs, one tcp, one utp.
		Addresses: Addresses{
			Swarm: []string{
				"/ip4/0.0.0.0/tcp/4001",
				// "/ip4/0.0.0.0/udp/4002/utp", // disabled for now.
				"/ip6/::/tcp/4001",
			},
			API:     "/ip4/127.0.0.1/tcp/5001",
			Gateway: "/ip4/127.0.0.1/tcp/8080",
		},

		Datastore: datastore,
		Bootstrap: BootstrapPeerStrings(bootstrapPeers),
		Identity:  identity,
		Discovery: Discovery{MDNS{
			Enabled:  true,
			Interval: 10,
		}},

		// setup the node mount points.
		Mounts: Mounts{
			IPFS: "/ipfs",
			IPNS: "/ipns",
		},

		Ipns: Ipns{
			ResolveCacheSize: 128,
		},

		Gateway: Gateway{
			RootRedirect: "",
			Writable:     false,
			PathPrefixes: []string{},
			HTTPHeaders: map[string][]string{
				"Access-Control-Allow-Origin":  []string{"*"},
				"Access-Control-Allow-Methods": []string{"GET"},
				"Access-Control-Allow-Headers": []string{"X-Requested-With"},
			},
		},
		Reprovider: Reprovider{
			Interval: "12h",
		},
	}

	return conf, nil
}

func datastoreConfig() (Datastore, error) {
	dspath, err := DataStorePath("")
	if err != nil {
		return Datastore{}, err
	}
	return Datastore{
		Path:               dspath,
		Type:               "leveldb",
		StorageMax:         "10GB",
		StorageGCWatermark: 90, // 90%
		GCPeriod:           "1h",
		HashOnRead:         false,
		BloomFilterSize:    0,
	}, nil
}

// identityConfig initializes a new identity.
func identityConfig(out io.Writer, nbits int) (Identity, error) {
	// TODO guard higher up
	ident := Identity{}
	if nbits < 1024 {
		return ident, errors.New("Bitsize less than 1024 is considered unsafe.")
	}

	fmt.Fprintf(out, "generating %v-bit RSA keypair...", nbits)
	sk, pk, err := ci.GenerateKeyPair(ci.RSA, nbits)
	if err != nil {
		return ident, err
	}
	fmt.Fprintf(out, "done\n")

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	skbytes, err := sk.Bytes()
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)

	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return ident, err
	}
	ident.PeerID = id.Pretty()

/////cmd := exec.Command("sudo","cp","/sys/class/dmi/id/product_uuid","/opt/iservstor/data/.uuid.txt")
cmd := exec.Command("sudo","cp","/proc/sys/kernel/random/uuid","/opt/iservstor/data/.uuid.txt")
cmd.Run()
cmd2 := exec.Command("sudo","chmod","0777","/opt/iservstor/data/.uuid.txt")
cmd2.Run()
Estring := base64.StdEncoding.EncodeToString([]byte(GetDomainName()+GetUuid()))/////
// Should readfile here
ident.GroupID = Estring
/////ident.GroupID = "iServDB"+GetUuid()

	fmt.Fprintf(out, "peer identity: %s\n", ident.PeerID)
	return ident, nil
}

func GetUuid() string{
        inputFile, Error := os.Open("/opt/iservstor/data/.uuid.txt")
        if Error != nil {
                fmt.Println("get uuid error !!")
                return "NO FILE"
        }
        defer inputFile.Close()
        inputReader := bufio.NewReader(inputFile)
        inputString, Error := inputReader.ReadString('\n')
        if Error == io.EOF {
                return "NO CONTENT"
        }
        return strings.Replace(inputString,"\n","",-1)
}

func GetDomainName() string{
	dat, err := ioutil.ReadFile("/opt/iservstor/conf/iservstor.conf")
	if err != nil{
		fmt.Println("get domain name error !!")
		return "NO FILE"
	}
	 tmp := strings.Split(string(dat),"DOMAIN_NAME")[1]
	 tmpp := strings.Split(tmp,"\n")[0]
	 tmppp := strings.Split(tmpp,"=")[1]
	 DomainName := strings.Replace(tmppp," ","",-1)
	 fmt.Printf("\nKEVIN DOMAIN NAME : %s\n",DomainName)
	 return DomainName
/*
	inputFile, Error := os.Open("/tmp/.config.txt")
	if Error != nil {
		fmt.Println("error !!")
		return "NO FILE"
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	inputString, Error := inputReader.ReadString('\n')
	if Error == io.EOF {
		return "NO CONTENT"
	}
	fmt.Printf("\n\nKEVIN GetUUID Test : %s 123123\n\n",inputReader.ReadString('\n'));
	return strings.Replace(inputString,"\n","",-1)
*/
}
