package server

import (
	"context"
	"log"
	"net"
	"strconv"

	"github.com/charmbracelet/wish"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// #nosec G101
const (
	PreSharedUser     = "kenobi"
	PresharedPassword = "8bbdc6746352ca9cdd6d7a662fa76d118a02774d8b85f0033e63b7a19ada3899"
)

type Config struct {
	User    string
	Listen  string
	Port    int
	NoShell bool
}

type Server struct {
	ssh *ssh.Server
	cfg *Config
}

func passwordHandler(ctx ssh.Context, password string) bool {
	return ctx.User() == PreSharedUser && password == PresharedPassword
}

func New(cfg *Config) (*Server, error) {
	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(cfg.Listen, strconv.Itoa(cfg.Port))),
		wish.WithHostKeyPath(".ssh/revish_ed25519"),
		wish.WithPasswordAuth(passwordHandler),
		wish.WithMiddleware(
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	srv.ReversePortForwardingCallback = newReversePortForwardingCallback()
	forwardHandler := &ssh.ForwardedTCPHandler{}
	srv.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        forwardHandler.HandleSSHRequest,
		"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
	}
	srv.ChannelHandlers = map[string]ssh.ChannelHandler{
		"direct-tcpip":                 ssh.DirectTCPIPHandler,
		"session":                      ssh.DefaultSessionHandler,
		ChannelRegisterRemoteSession:   registerRemoteSession(),
		ChannelUnregisterRemoteSession: unregisterRemoteSession(),
		//ChannelTryRemotePort:           func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {}, // TODO: implement
	}
	srv.SessionRequestCallback = newSessionRequestCallback(cfg.NoShell)

	s := &Server{
		ssh: srv,
		cfg: cfg,
	}

	return s, nil
}

type Channel string

const (
	ChannelRegisterRemoteSession   = "register_remote_session"
	ChannelUnregisterRemoteSession = "unregister_remote_session"
	ChannelTryRemotePort           = "request_remote_port"
)

func (s *Server) Serve() error {
	return s.ssh.ListenAndServe()
}

func (s *Server) Close(ctx context.Context) error {
	return s.ssh.Shutdown(ctx)
}

func newSessionRequestCallback(forbidden bool) ssh.SessionRequestCallback {
	return func(sess ssh.Session, requestType string) bool {
		if forbidden {
			log.Printf("denied session request from %s %s", sess.User(), sess.RemoteAddr().String())
			return false
		}
		return true
	}
}

func newReversePortForwardingCallback() ssh.ReversePortForwardingCallback {
	return func(ctx ssh.Context, host string, port uint32) bool {
		log.Printf("Attempt to bind at %s:%d granted", host, port)
		return true
	}
}

type ExtraInfo struct {
	CurrentUser      string
	Hostname         string
	ListeningAddress string
}

func registerRemoteSession() ssh.ChannelHandler {
	return func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
		var extraInfo ExtraInfo
		err := gossh.Unmarshal(newChan.ExtraData(), &extraInfo)
		newChan.Reject(gossh.Prohibited, "registered_remote_session")
		if err != nil {
			log.Printf("Could not parse extra info from %s", conn.RemoteAddr())
			return
		}
		log.Printf(
			"New remote connection from %s: %s on %s reachable via %s",
			conn.RemoteAddr(),
			extraInfo.CurrentUser,
			extraInfo.Hostname,
			extraInfo.ListeningAddress,
		)
	}
}

func unregisterRemoteSession() ssh.ChannelHandler {
	return func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
		var extraInfo ExtraInfo
		err := gossh.Unmarshal(newChan.ExtraData(), &extraInfo)
		newChan.Reject(gossh.Prohibited, "unregistered_remote_session")
		if err != nil {
			log.Printf("Could not parse extra info from %s", conn.RemoteAddr())
			return
		}
		log.Printf(
			"Close remote connection from %s: %s on %s reachable via %s",
			conn.RemoteAddr(),
			extraInfo.CurrentUser,
			extraInfo.Hostname,
			extraInfo.ListeningAddress,
		)
	}
}
