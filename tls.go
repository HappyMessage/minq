package minq

import (
	"encoding/hex"
	"fmt"
	"github.com/bifurcation/mint"
)

type TlsConfig struct {
}

func (c TlsConfig) toMint() *mint.Config {
	// TODO(ekr@rtfm.com): Provide a real config
	return &mint.Config{ServerName: "example.com", NonBlocking: true}
}

type tlsConn struct {
	conn     *connBuffer
	tls      *mint.Conn
	finished bool
	cs       *mint.CipherSuiteParams
}

func newTlsConn(conf TlsConfig, role uint8) *tlsConn {
	isClient := true
	if role == RoleServer {
		isClient = false
	}

	c := newConnBuffer()

	return &tlsConn{
		c,
		mint.NewConn(c, conf.toMint(), isClient),
		false,
		nil,
	}
}

func (c *tlsConn) handshake(input []byte) ([]byte, error) {
	logf(logTypeTls, "TLS handshake input len=%v", len(input))
	logf(logTypeTrace, "TLS handshake input = %v", hex.EncodeToString(input))
	if input != nil {
		err := c.conn.input(input)
		if err != nil {
			return nil, err
		}
	}
	assert(c.conn.OutputLen() == 0)
	alert := c.tls.Handshake()

	switch alert {
	case mint.AlertNoAlert:
		logf(logTypeTls, "TLS handshake complete")
		cs := c.tls.GetConnectionState().CipherSuite
		c.cs = &cs
		c.finished = true
	case mint.AlertWouldBlock:
		logf(logTypeTls, "TLS would have blocked")
	default:
		return nil, fmt.Errorf("TLS sent an alert %v", alert)
	}
	logf(logTypeTls, "TLS wrote %d bytes", c.conn.OutputLen())

	return c.conn.getOutput(), nil
}
