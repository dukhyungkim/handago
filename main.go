package main

import (
	"handago/config"
	"handago/handler"
	"handago/stream"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	log.Println("Handago Starting.")

	opts, err := config.ParseFlags()
	if err != nil {
		log.Panicln(err)
	}

	if !opts.Shared && opts.Company == "" && !opts.Internal {
		log.Println("--shared, --company or --internal must be set")
		return
	}

	if opts.Company != "" && opts.Adapter == "" {
		log.Println("--adapter must be set")
		return
	}

	cfg, err := config.NewConfig(opts)
	if err != nil {
		log.Panicln(err)
	}

	streamClient, err := stream.NewStreamClient(cfg.Nats)
	if err != nil {
		log.Panicln(err)
	}
	defer streamClient.Close()
	log.Println("connect to nats ... success")

	dockerHandler, err := handler.New(opts, cfg.Etcd, streamClient)
	if err != nil {
		log.Panicln(err)
	}
	defer dockerHandler.Close()
	log.Println("setup DockerHandler ... success")

	if opts.Shared {
		err = streamClient.ClamSharedAction(dockerHandler.HandleUpDownAction)
		if err != nil {
			log.Panicln(err)
		}
	}

	if opts.Company != "" {
		err = streamClient.ClamCompanyAction(opts.Company, dockerHandler.HandleUpDownAction)
		if err != nil {
			log.Panicln(err)
		}
	}

	if opts.Internal {
		err = streamClient.ClamInternalAction(dockerHandler.HandleUpDownAction)
		if err != nil {
			log.Panicln(err)
		}
	}

	waitSignal()
}

func waitSignal() {
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	<-sigterm
	log.Println("terminating: via signal")
}
