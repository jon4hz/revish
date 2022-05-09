package revishlist

import (
	"bytes"
	"sync"

	"github.com/charmbracelet/wish"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wishlist"
	"github.com/gliderlabs/ssh"
)

type Server struct {
	mu          sync.Mutex
	sessionChan chan int
	cfg         *wishlist.Config
	endpoints   []*Endpoint
}

type Endpoint struct {
	SessionID        []byte
	RemoteAddr       string
	User             string
	Hostname         string
	ListeningAddress string
}

func (e *Endpoint) Equal(other *Endpoint) bool {
	return bytes.Equal(e.SessionID, other.SessionID)
}

func (e *Endpoint) Wishlist() *wishlist.Endpoint {
	return &wishlist.Endpoint{
		Address: e.ListeningAddress,
		User:    "jonah",
		Name:    e.Hostname,
	}
}

func New() *Server {
	s := new(Server)
	s.cfg = &wishlist.Config{
		Listen: "localhost",
		Port:   4444,
		//Users:        []wishlist.User{{Name: "jonah"}},
		Factory: func(e wishlist.Endpoint) (*ssh.Server, error) {
			return wish.NewServer(
				wish.WithAddress(e.Address),
				wish.WithHostKeyPath(".ssh/revish_wishlist_ed25519"),
				wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
					return true
				}),
				wish.WithMiddleware(
					append(
						e.Middlewares,
						lm.Middleware(),
					)...,
				),
			)
		},
		EndpointChan: make(chan []*wishlist.Endpoint),
	}
	return s
}

func (s *Server) ListenAndServe() error {
	return wishlist.Serve(s.cfg)
}

func (s *Server) AddEndpoint(e *Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoints = append(s.endpoints, e)
	s.broadcastNewEndpoints()
}

func (s *Server) RemoveEndpoint(e *Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.broadcastNewEndpoints()
	for i, ep := range s.endpoints {
		if ep.Equal(e) {
			s.endpoints = append(s.endpoints[:i], s.endpoints[i+1:]...)
			return
		}
	}
}

func (s *Server) broadcastNewEndpoints() {
	e := s.WishlistEndpoints()
	s.cfg.Endpoints = e
	s.cfg.EndpointChan <- e
}

func (s *Server) WishlistEndpoints() []*wishlist.Endpoint {
	we := make([]*wishlist.Endpoint, 0, len(s.endpoints))
	for _, e := range s.endpoints {
		we = append(we, e.Wishlist())
	}
	return we
}
