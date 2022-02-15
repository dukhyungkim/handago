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
		log.Fatalln(err)
	}

	if !opts.Shared && opts.Company == "" {
		log.Fatalln("--shared or --company must be set")
	}

	cfg, err := config.NewConfig(opts)
	if err != nil {
		log.Fatalln(err)
	}

	streamClient, err := stream.NewStreamClient(cfg.Nats)
	if err != nil {
		log.Fatalln(err)
	}
	defer streamClient.Close()
	log.Println("connect to nats ... success")

	dockerHandler, err := handler.NewDockerHandler(cfg.Etcd, streamClient)
	if err != nil {
		log.Panicln(err)
	}
	defer dockerHandler.Close()
	log.Println("setup DockerHandler ... success")

	if opts.Shared {
		if err := streamClient.ClamSharedAction(dockerHandler.HandleAction); err != nil {
			log.Panicln(err)
		}
	}

	if opts.Company != "" {
		if err := streamClient.ClamCompanyAction(opts.Company, opts.Host, dockerHandler.HandleCompanyAction); err != nil {
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
