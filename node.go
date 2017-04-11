package sodibus

import "net"
import "log"
import "sync"
import "math/rand"
import "github.com/Unknwon/com"
import "github.com/sodibus/packet"

// locate a Callee across mutiple nodes

type CalleeId struct {
	NodeId uint64
	ClientId uint64
}

type Node struct {
	// node information
	id uint64
	addr string
	listener *net.TCPListener
	// connections
	lastConnId uint64
	conns map[uint64]*Conn
	connsLock *sync.RWMutex
}

func NewNode(addr string) *Node {
	return &Node {
		id: rand.Uint64(),
		addr: addr,
		conns: make(map[uint64]*Conn),
		connsLock: &sync.RWMutex{},
	}
}

// Conn Management

func (n *Node) NewConnId() uint64 {
	n.lastConnId = n.lastConnId + 1
	return n.lastConnId
}

// Loops

func (n *Node) Run() error {
	// resolve TCP address to bind
	tcpAddr, err := net.ResolveTCPAddr("tcp", n.addr)
	if err != nil { return err }

	// create listener
	n.listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil { return err }

	log.Println("SODIBus", n.id, "listening at", n.addr)

	// accepting
	for {
		// accept
		conn, err := n.listener.AcceptTCP()
		if err == nil {
			// create client, auto atomical id
			c := NewConn(conn, n.NewConnId())
			// start Conn
			go c.Run(n)
		} else {
			log.Fatal("Failed to accept", err)
			return err
		}
	}
}

// Resolving

func (n *Node) ResolveCallee(name string) *CalleeId {
	var calleeId *CalleeId
	// find a usable client and send back
	n.connsLock.RLock()
	for _, v := range n.conns {
		if v.isCallee && com.IsSliceContainsStr(v.provides, name) {
			calleeId = &CalleeId{
				NodeId: n.id,
				ClientId: v.id,
			}
			break
		}
	}
	n.connsLock.RUnlock()
	// send nil if nothing found
	return calleeId
}

// ConnHandler

func (n *Node) ConnDidStart(c *Conn) {
	log.Println("New Conn: id =", c.id, ", callee =", c.isCallee, ", provides =", c.provides)
	// put to internal registry
	n.connsLock.Lock()
	n.conns[c.id] = c
	n.connsLock.Unlock()
}

func (n *Node) ConnDidReceiveFrame(c *Conn, f *packet.Frame) {
}

func (n *Node) ConnWillClose(c *Conn) {
	log.Println("Lost Conn: id =", c.id, ", callee =", c.isCallee)
	// remove from internal registry
	n.connsLock.Lock()
	delete(n.conns, c.id)
	n.connsLock.Unlock()
}
