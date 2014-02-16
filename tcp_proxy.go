package proxy

import (
	"github.com/crosbymichael/log"
	"net"
)

func newTcpPRoxy(host *Host, backend *Backend) (*tcpProxy, error) {
	return &tcpProxy{
		host:    host,
		backend: backend,
	}, nil
}

type tcpProxy struct {
	backend  *Backend
	host     *Host
	listener *net.TCPListener
}

func (p *tcpProxy) Close() error {
	return p.listener.Close()
}

func (p *tcpProxy) Run(handler Handler) (err error) {
	p.listener, err = net.ListenTCP("tcp", &net.TCPAddr{IP: p.backend.IP, Port: p.backend.Port})
	if err != nil {
		return err
	}
	defer p.Close()

	var (
		errorCount  int
		connections = make(chan *net.TCPConn, p.backend.ConnectionBuffer)
	)

	for i := 0; i < p.backend.MaxConcurrent; i++ {
		go proxyWorker(connections, p.backend, p.host.Dns, handler)
	}

	for {
		conn, err := p.listener.AcceptTCP()
		if err != nil {
			errorCount++
			if errorCount > p.host.MaxListenErrors {
				return err
			}
			log.Logf(log.ERROR, "tcp accept error %s", err)
			continue
		}
		connections <- conn
	}
	return nil
}

func proxyWorker(c chan *net.TCPConn, backend *Backend, dns string, handler Handler) {
	for conn := range c {
		if err := handler.HandleConn(conn); err != nil {
			log.Logf(log.ERROR, "handle connection %s", err)
		}
	}
}
