package commands

import (
	"bytes"
	"testing"
)

func TestPrintDiagnostics(t *testing.T) {
	output := DiagnosticOutput{
		Peers: []DiagnosticPeer{
			{ID: "QmNrjRuUtBNZAigzLRdZGN1YCNUxdF2WY2HnKyEFJqoTeg",
				UptimeSeconds: 14,
				Connections: []DiagnosticConnection{
					{ID: "QmNrjRuUtBNZAigzLRdZGN1YCNUxdF2WY2HnKyEFJqoTeg",
						NanosecondsLatency: 1347899,
					},
				},
			},
			{ID: "QmUaUZDp6QWJabBYSKfiNmXLAXD8HNKnWZh9Zoz6Zri9Ti",
				UptimeSeconds: 14,
			},
		},
	}
	buf := new(bytes.Buffer)
	if err := printDiagnostics(buf, &output); err != nil {
		t.Fatal(err)
	}
	t.Log(buf.String())
}
