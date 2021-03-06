package minq

import (
	"net"
)

// TransportFactory makes transports bound to a specific remote
// address.
type TransportFactory interface {
	// Make a transport object bound to |remote|.
	makeTransport(remote *net.UDPAddr) (Transport, error)
}

// Server represents a QUIC server. A server can be fed an arbitrary
// number of packets and will create Connections as needed, passing
// each packet to the right connection.
type Server struct {
	handler      ServerHandler
	transFactory TransportFactory
	tls          TlsConfig
	addrTable    map[string]*Connection
	idTable      map[ConnectionId]*Connection
}

// Interface for the handler object which the Server will call
// to notify of events.
type ServerHandler interface {
	// A new connection has been created and can be found in |c|.
	NewConnection(c *Connection)
}

// Pass an incoming packet to the Server.
func (s *Server) Input(addr *net.UDPAddr, data []byte) (*Connection, error) {
	logf(logTypeServer, "Received packet from %v", addr)
	var hdr packetHeader

	_, err := decode(&hdr, data)
	if err != nil {
		return nil, err
	}

	var conn *Connection

	if hdr.hasConnId() {
		logf(logTypeServer, "Received conn id %v", hdr.ConnectionID)
		conn = s.idTable[hdr.ConnectionID]
		if conn != nil {
			logf(logTypeServer, "Found by conn id")
		}
	}

	if conn == nil {
		conn = s.addrTable[addr.String()]
	}

	if conn == nil {
		logf(logTypeServer, "New server connection from addr %v", addr)
		trans, err := s.transFactory.makeTransport(addr)
		if err != nil {
			return nil, err
		}
		conn = NewConnection(trans, RoleServer, s.tls, nil)
		s.idTable[conn.serverConnId] = conn
		s.addrTable[addr.String()] = conn

		if s.handler != nil {
			s.handler.NewConnection(conn)
		}
	}

	return conn, conn.Input(data)
}

// Create a new QUIC server with the provide TLS config.
func NewServer(factory TransportFactory, tls TlsConfig, handler ServerHandler) *Server {
	return &Server{
		handler,
		factory,
		tls,
		make(map[string]*Connection),
		make(map[ConnectionId]*Connection),
	}
}
