package main

import (
	"io"
	"log"
	"net"
)

// Proxy - Manages a Proxy connection, piping data between local and remote.
type Proxy struct {
	sentBytes     uint64
	receivedBytes uint64
	laddr, raddr  *net.TCPAddr
	lconn, rconn  io.ReadWriteCloser
	erred         bool
	errsig        chan bool

	// Settings
	Nagles    bool
	Prefix    string
	OnMessage ProxyOnMessage
}

type ProxyOnMessage func(bool, []byte)

// New - Create a new Proxy instance. Takes over local connection passed in,
// and closes it when finished.
func NewProxy(lconn *net.TCPConn, laddr, raddr *net.TCPAddr, onmsg ProxyOnMessage) *Proxy {
	return &Proxy{
		lconn:     lconn,
		laddr:     laddr,
		raddr:     raddr,
		erred:     false,
		errsig:    make(chan bool),
		OnMessage: onmsg,
	}
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

func (p *Proxy) WriteDownStream(b []byte) {
	// log.Printf("%s<<< %d %x\n", p.Prefix, len(b), b)
	p.lconn.Write(b)
}

// Start - open connection to remote and start proxying data.
func (p *Proxy) Start() {
	defer p.lconn.Close()

	var err error
	p.rconn, err = net.DialTCP("tcp", nil, p.raddr)
	if err != nil {
		log.Printf("Remote connection failed: %s\n", err)
		return
	}
	defer p.rconn.Close()

	//nagles?
	if p.Nagles {
		if conn, ok := p.lconn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
		if conn, ok := p.rconn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
	}

	//display both ends
	log.Printf("Opened %s >>> %s\n", p.laddr.String(), p.raddr.String())

	//bidirectional copy
	go p.pipe(p.lconn, p.rconn)
	go p.pipe(p.rconn, p.lconn)

	//wait for close...
	<-p.errsig
	log.Printf("Closed (%d bytes sent, %d bytes recieved)\n", p.sentBytes, p.receivedBytes)
}

func (p *Proxy) err(s string, err error) {
	if p.erred {
		return
	}
	if err != io.EOF {
		log.Printf(s, err)
	}
	p.errsig <- true
	p.erred = true
}

func (p *Proxy) pipe(src, dst io.ReadWriter) {
	islocal := src == p.lconn
/*
	var dataDirection string
	if islocal {
		dataDirection = "%s>>> %d %x\n"
	} else {
		dataDirection = "%s<<< %d %x\n"
	}
*/
	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed '%s'\n", err)
			return
		}
		b := buff[:n]

		//show output
		// log.Printf(dataDirection, p.Prefix, n, b)

		p.OnMessage(islocal, b)

		//write out result
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed '%s'\n", err)
			return
		}
		if islocal {
			p.sentBytes += uint64(n)
		} else {
			p.receivedBytes += uint64(n)
		}
	}
}
