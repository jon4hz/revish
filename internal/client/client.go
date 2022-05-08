package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"strings"
	"syscall"

	"github.com/charmbracelet/wish"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/jon4hz/revish/internal/middleware/shell"
	"github.com/jon4hz/revish/internal/server"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Config struct {
	Server     string
	RemotePort int
	Listen     string
	Port       int
	Shell      string
	Quiet      bool
}

type Client struct {
	ssh     *ssh.Server
	gossh   *gossh.Client
	ln      net.Listener
	cfg     *Config
	address string
}

func New(cfg *Config) (*Client, error) {
	srv, err := wish.NewServer(
		wish.WithHostKeyPath(".ssh/revish_client_ed25519"),
		wish.WithAuthorizedKeys(".authorized_keys"),
		wish.WithMiddleware(
			lm.Middleware(),
			shell.Middleware(cfg.Shell),
		),
	)
	if err != nil {
		return nil, err
	}

	c := &Client{
		ssh: srv,
		cfg: cfg,
	}
	gc, err := c.newGoSSHClient(server.PreSharedUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create go ssh client: %s", err)
	}
	c.gossh = gc

	return c, nil
}

func (c *Client) Serve() error {
	c.registerNewClient()
	return c.ssh.Serve(c.ln)
}

func (c *Client) Close(ctx context.Context) error {
	if err := c.stopRemoteSession(); err != nil {
		return err
	}
	return c.ssh.Shutdown(ctx)
}

func (c *Client) newGoSSHClient(username string) (*gossh.Client, error) {
	config := &gossh.ClientConfig{
		User: username,
		Auth: []gossh.AuthMethod{
			gossh.Password(server.PresharedPassword),
		},
		// #nosec G106
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	log.Printf("Dialling home via ssh to %s", c.cfg.Server)
	return tryPassword(c.cfg.Server, config)
}

func (c *Client) registerNewClient() error {
	ln, err := c.negotiateRemotePort()
	if err != nil {
		return err
	}
	log.Printf("Success: listening at home on %s", c.address)
	c.ln = ln
	return c.createRemoteSession()
}

// TODO: implement multiple retries
func (c *Client) negotiateRemotePort() (net.Listener, error) {
	ln, err := c.gossh.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	c.address = ln.Addr().String()
	log.Printf("Trying to listen on %s", c.address)
	if ok, err := c.tryRemotePort(); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("failed to negotiate remote port")
	}
	return ln, nil
}

func tryPassword(address string, config *gossh.ClientConfig) (*gossh.Client, error) {
	maxRetries := 3
	for i := 0; i < maxRetries+1; i++ {
		if i == maxRetries {
			return nil, fmt.Errorf("Could not connect to %s", address)
		}
		client, err := gossh.Dial("tcp", address, config)
		if err == nil {
			return client, nil
		} else if strings.HasSuffix(err.Error(), "no supported methods remain") {
			fmt.Printf("Enter password: ")
			data, err := term.ReadPassword(syscall.Stdin)
			fmt.Println()
			if err != nil {
				log.Println(err)
				continue
			}

			config.Auth = []gossh.AuthMethod{
				gossh.Password(string(data)),
			}
		} else {
			return nil, err
		}
	}
	return nil, fmt.Errorf("Could not connect to %s", address)
}

func (c *Client) tryRemotePort() (bool, error) {
	extraInfo := getExtraInfo(c.address)
	newChan, newReq, err := c.gossh.OpenChannel(server.ChannelTryRemotePort, gossh.Marshal(&extraInfo))
	if err != nil && !strings.Contains(err.Error(), "accepted") {
		return true, nil
	} else if err != nil {
		return false, err
	}
	if err == nil {
		go gossh.DiscardRequests(newReq)
		newChan.Close()
	}
	return false, nil
}

func (c *Client) createRemoteSession() error {
	extraInfo := getExtraInfo(c.address)
	newChan, newReq, err := c.gossh.OpenChannel(server.ChannelRegisterRemoteSession, gossh.Marshal(&extraInfo))
	if err != nil && !strings.Contains(err.Error(), "registered_remote_session") {
		return fmt.Errorf("error: could not create info channel: %+v", err)
	}

	if err == nil {
		go gossh.DiscardRequests(newReq)
		newChan.Close()
	}
	return nil
}

func (c *Client) stopRemoteSession() error {
	extraInfo := getExtraInfo(c.address)
	newChan, newReq, err := c.gossh.OpenChannel(server.ChannelUnregisterRemoteSession, gossh.Marshal(&extraInfo))
	if err != nil && !strings.Contains(err.Error(), "unregistered_remote_session") {
		log.Printf("error: could not create info channel: %+v", err)
	}
	if err == nil {
		go gossh.DiscardRequests(newReq)
		newChan.Close()
	}
	return nil
}

func getExtraInfo(listeningAddress string) server.ExtraInfo {
	extraInfo := server.ExtraInfo{ListeningAddress: listeningAddress}
	if usr, err := user.Current(); err != nil {
		extraInfo.CurrentUser = "ERROR"
	} else {
		extraInfo.CurrentUser = usr.Username
	}
	if hostname, err := os.Hostname(); err != nil {
		extraInfo.Hostname = "ERROR"
	} else {
		extraInfo.Hostname = hostname
	}
	return extraInfo
}
