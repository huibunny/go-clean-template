package main

import (
	"flag"
	"log"

	"github.com/evrone/go-clean-template/config"
	"github.com/evrone/go-clean-template/internal/app"
	consulutil "github.com/huibunny/gocore/thirdpart/consul"
)

func main() {
	// config args, priority: config > consul
	var (
		configFile   = flag.String("config", "", "config file, prior to use.")
		consulAddr   = flag.String("consul", "localhost:8500", "consul server address.")
		consulFolder = flag.String("folder", "", "consul kv folder.")
		serviceName  = flag.String("name", "microapp", "both microservice name and kv name.")
		listenAddr   = flag.String("listen", ":8080", "listen address.")
	)
	flag.Parse()
	// Configuration
	cfg := &config.Config{}
	var err error
	if len(*configFile) > 0 {
		cfg, err = config.NewConfig(*configFile)
	} else if len(*consulAddr) > 0 {
		consulClient, serviceID, port, err := consulutil.RegisterAndCfgConsul(cfg, *consulAddr, *serviceName, *listenAddr, *consulFolder)
		if err != nil {
			log.Fatalf("fail to register consul: %v.", err)
		}
		defer consulutil.DeregisterService(consulClient, serviceID)
		app.Run(cfg, port)
	} else {
		log.Fatalf("no input: config file or consul address not provided!")
		return
	}

	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
}
