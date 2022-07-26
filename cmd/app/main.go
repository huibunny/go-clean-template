package main

import (
	"flag"
	"log"

	"github.com/evrone/go-clean-template/config"
	"github.com/evrone/go-clean-template/internal/app"
	consulapi "github.com/hashicorp/consul/api"
)

func main() {
	// config args, priority: config > consul
	var (
		configFile  = flag.String("config", "", "config file, prior to use.")
		consulAddr  = flag.String("consul", "localhost:8500", "consul server address.")
		serviceName = flag.String("name", "microapp", "both microservice name and kv name.")
		listenAddr  = flag.String("listen", ":8080", "listen address.")
	)
	flag.Parse()
	host, port := app.GetHostPort(*listenAddr)
	// Configuration
	var cfg *config.Config
	var err error
	if len(*configFile) > 0 {
		cfg, err = config.NewConfig(*configFile)
	} else if len(*consulAddr) > 0 {
		var serviceID string
		var consulClient *consulapi.Client
		cfg, consulClient, serviceID, err = app.RegisterAndCfgConsul(*consulAddr, *serviceName, host, port)
		defer consulClient.Agent().ServiceDeregister(serviceID)
	} else {
		log.Fatalf("no input: config file or consul address not provided!")
		return
	}

	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg, port)
}
