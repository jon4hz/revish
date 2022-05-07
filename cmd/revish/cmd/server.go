package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jon4hz/revish/interal/server"
	"github.com/muesli/coral"
)

var serverFlags server.Config

var serverCmd = &coral.Command{
	Use:   "server",
	Short: "Start the server",
	RunE:  runServer,
}

func init() {
	serverCmd.Flags().StringVarP(&serverFlags.User, "user", "u", "", "user allow to connect to the server")
	serverCmd.Flags().StringVarP(&serverFlags.Listen, "listen", "l", "", "address to listen on")
	serverCmd.Flags().IntVarP(&serverFlags.Port, "port", "p", 4444, "port to listen on")
	serverCmd.Flags().BoolVarP(&serverFlags.NoShell, "no-shell", "N", false, "deny all incoming shell/exec/subsystem and local port forwarding requests")
}

func runServer(cmd *coral.Command, args []string) error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv, err := server.New(&serverFlags)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		log.Printf("Starting SSH server on %s:%d", serverFlags.Listen, serverFlags.Port)
		if err := srv.Serve(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := srv.Close(ctx); err != nil {
		log.Fatalln(err)
	}
	return nil
}
