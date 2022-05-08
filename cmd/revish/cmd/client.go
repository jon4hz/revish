package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jon4hz/revish/internal/client"
	"github.com/muesli/coral"
)

var clientFlags client.Config

var clientCmd = &coral.Command{
	Use:   "client",
	Short: "Start the client",
	RunE:  runClient,
	Args:  coral.ExactArgs(1),
}

func init() {
	clientCmd.Flags().StringVarP(&clientFlags.Listen, "listen", "l", "127.0.0.1", "address to listen on")
	clientCmd.Flags().IntVarP(&clientFlags.Port, "port", "p", 0, "port to listen on")
	clientCmd.Flags().StringVarP(&clientFlags.Shell, "shell", "s", "/bin/bash", "shell to attach to")
	clientCmd.Flags().BoolVarP(&clientFlags.Quiet, "quiet", "q", false, "disable verbose output")
}

func runClient(cmd *coral.Command, args []string) error {
	clientFlags.Server = args[0]

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	c, err := client.New(&clientFlags)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		log.Printf("Starting SSH server on %s:%d", clientFlags.Listen, clientFlags.Port)
		if err := c.Serve(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.Close(ctx); err != nil {
		log.Fatalln(err)
	}
	return nil
}
