// Package app configures and runs application.
package app

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"

	"github.com/evrone/go-clean-template/config"
	amqprpc "github.com/evrone/go-clean-template/internal/controller/amqp_rpc"
	v1 "github.com/evrone/go-clean-template/internal/controller/http/v1"
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/internal/usecase/repo"
	"github.com/evrone/go-clean-template/internal/usecase/webapi"
	"github.com/evrone/go-clean-template/pkg/httpserver"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/evrone/go-clean-template/pkg/postgres"
	"github.com/evrone/go-clean-template/pkg/rabbitmq/rmq_rpc/server"
	consulapi "github.com/hashicorp/consul/api"
)

// Run creates objects via constructors.
func Run(cfg *config.Config, port string) {
	l := logger.New(cfg.Log.Level)

	// Repository
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal(fmt.Errorf("app - Run - postgres.New: %w", err))
	} else {
		l.Info("app - Run - postgres.")
	}
	defer pg.Close()

	// Use case
	translationUseCase := usecase.New(
		repo.New(pg),
		webapi.New(),
	)
	// HTTP Server
	handler := gin.New()
	v1.NewRouter(handler, l, translationUseCase)
	httpServer := httpserver.New(handler, httpserver.Port(port))
	l.Info("app - Run - httpServer: " + port + ".")

	// RabbitMQ RPC Server
	var rmqServer *server.Server
	if len(cfg.RMQ.URL) > 0 {
		rmqRouter := amqprpc.NewRouter(translationUseCase)

		rmqServer, err = server.New(cfg.RMQ.URL, cfg.RMQ.ServerExchange, rmqRouter, l)
		if err != nil {
			l.Fatal(fmt.Errorf("app - Run - rmqServer - server.New: %w", err))
		} else {
			l.Info("app - Run - rmqServer - server: " + cfg.RMQ.URL + ".")
		}
	} else {
		//
	}

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
		// case err = <-rmqServer.Notify():
		// 	l.Error(fmt.Errorf("app - Run - rmqServer.Notify: %w", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}

	if rmqServer != nil {
		err = rmqServer.Shutdown()
		if err != nil {
			l.Error(fmt.Errorf("app - Run - rmqServer.Shutdown: %w", err))
		}
	}
}

func RegisterAndCfgConsul(consulAddr string, serviceName string, host string, port string, consulInterval string, consulTimeout string) (*config.Config, *consulapi.Client, string, error) {
	// 创建consul api客户端
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = consulAddr
	consulClient, err := consulapi.NewClient(consulConfig)
	if err != nil {
		os.Exit(1)
	}

	cfg := &config.Config{}
	var serviceID string
	serviceID, err = registerService(serviceName, *consulClient, host, port, consulInterval, consulTimeout)
	if err == nil {
		kv, _, err := consulClient.KV().Get(serviceName, nil)
		if err == nil {
			// only support yaml kv
			err = yaml.NewDecoder(strings.NewReader(string(kv.Value))).Decode(cfg)
			if err == nil {
			} else {
				print("error: " + err.Error())
			}
		} else {
			print("error: " + err.Error())
		}
	} else {
		print("error: " + err.Error())
	}
	return cfg, consulClient, serviceID, err
}

// RegisterService register service in consul
func registerService(service string, client consulapi.Client, svcHost string, svcPort string, consulInterval string, consulTimeout string) (string, error) {
	svcAddress := svcHost + ":" + svcPort

	// 设置Consul对服务健康检查的参数
	check := consulapi.AgentServiceCheck{
		HTTP:     "http://" + svcAddress + "/healthz",
		Interval: consulInterval + "s",
		Timeout:  consulTimeout + "s",
		Notes:    "Consul check service health status.",
	}

	port, _ := strconv.Atoi(svcPort)

	//设置微服务Consul的注册信息
	reg := &consulapi.AgentServiceRegistration{
		ID:      service + "_" + svcAddress,
		Name:    service,
		Address: svcHost,
		Port:    port,
		Check:   &check,
	}

	// 执行注册
	err := client.Agent().ServiceRegister(reg)

	return reg.ID, err
}

func GetHostPort(listenAddr string) (string, string) {
	host := "localhost"
	port := "0"
	hostPorts := strings.Split(listenAddr, ":")
	if len(hostPorts) > 1 {
		host = hostPorts[0]
		port = hostPorts[1]
	} else {
		port = hostPorts[0]
	}
	if len(host) == 0 {
		host = GetHostIP()
	} else {
	}

	return host, port
}

// GetHostIP get host ip address
func GetHostIP() string {
	hostAddress := ""
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {

		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				hostAddress = ipnet.IP.String()
				break
			}

		}
	}
	return hostAddress
}
