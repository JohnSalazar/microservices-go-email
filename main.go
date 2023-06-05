package main

import (
	"context"
	"email/src/routers"
	"email/src/services"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpc_proto "github.com/oceano-dev/microservices-go-common/grpc"
	common_grpc_client "github.com/oceano-dev/microservices-go-common/grpc/email/client"
	"github.com/oceano-dev/microservices-go-common/helpers"
	"github.com/oceano-dev/microservices-go-common/httputil"

	"github.com/oceano-dev/microservices-go-common/config"
	common_security "github.com/oceano-dev/microservices-go-common/security"
	common_services "github.com/oceano-dev/microservices-go-common/services"
	common_tasks "github.com/oceano-dev/microservices-go-common/tasks"
	provider "github.com/oceano-dev/microservices-go-common/trace/otel/jaeger"

	consul "github.com/hashicorp/consul/api"
	common_consul "github.com/oceano-dev/microservices-go-common/consul"
)

type Main struct {
	config       *config.Config
	emailService common_services.EmailService
	grpcServer   *grpc_proto.GrpcServer
	httpserver   httputil.HttpServer
	consulClient *consul.Client
	serviceID    string
}

func NewMain(
	config *config.Config,
	emailService common_services.EmailService,
	grpcServer *grpc_proto.GrpcServer,
	httpserver httputil.HttpServer,
	consulClient *consul.Client,
	serviceID string,
) *Main {
	return &Main{
		config:       config,
		emailService: emailService,
		grpcServer:   grpcServer,
		httpserver:   httpserver,
		consulClient: consulClient,
		serviceID:    serviceID,
	}
}

var production *bool
var disableTrace *bool

func main() {
	production = flag.Bool("prod", false, "use -prod=true to run in production mode")
	disableTrace = flag.Bool("disable-trace", false, "use disable-trace=true if you want to disable tracing completly")

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := startup(ctx)
	if err != nil {
		panic(err)
	}

	providerTracer, err := provider.NewProvider(provider.ProviderConfig{
		JaegerEndpoint: app.config.Jaeger.JaegerEndpoint,
		ServiceName:    app.config.Jaeger.ServiceName,
		ServiceVersion: app.config.Jaeger.ServiceVersion,
		Production:     *production,
		Disabled:       *disableTrace,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer providerTracer.Close(ctx)
	log.Println("Connected to Jaegger")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	lis, err := net.Listen("tcp", app.config.GrpcServer.Port)
	if err != nil {
		log.Fatalf("Failed to listener: %v", err)
	}
	defer lis.Close()

	grpcServer, err := app.grpcServer.CreateGrpcServer()
	if err != nil {
		log.Fatalln(err)
	}

	emailService := services.NewEmailServerGrpc(app.emailService)
	common_grpc_client.RegisterEmailServiceServer(grpcServer, emailService)

	grpc_prometheus.Register(grpcServer)
	grpc_prometheus.EnableHandlingTimeHistogram()

	go func() {
		log.Printf("GRPC server is listening on port: %v", app.config.GrpcServer.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	app.httpserver.RunTLSServer()

	<-done
	err = app.consulClient.Agent().ServiceDeregister(app.serviceID)
	if err != nil {
		log.Printf("consul deregister error: %s", err)
	}

	grpcServer.GracefulStop()
	log.Print("Servers Stopped")
	os.Exit(0)
}

func startup(ctx context.Context) (*Main, error) {
	config := config.LoadConfig(*production, "./config/")
	helpers.CreateFolder(config.Folders)

	consulClient, serviceID, err := common_consul.NewConsulClient(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	checkServiceName := common_tasks.NewCheckServiceNameTask()

	certificateServiceNameDone := make(chan bool)
	go checkServiceName.ReloadServiceName(
		ctx,
		config,
		consulClient,
		config.Certificates.ServiceName,
		common_consul.CertificatesAndSecurityKeys,
		certificateServiceNameDone)
	<-certificateServiceNameDone

	metricService, err := common_services.NewMetricsService(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	certificatesService := common_services.NewCertificatesService(config)
	managerCertificates := common_security.NewManagerCertificates(config, certificatesService)
	emailService := services.NewEmailService(config)

	checkCertificates := common_tasks.NewCheckCertificatesTask(config, managerCertificates, emailService)
	certsDone := make(chan bool)
	go checkCertificates.Start(ctx, certsDone)
	<-certsDone

	grpcServer := grpc_proto.NewGrpcServer(config, certificatesService, metricService)

	router := routers.NewRouter(config, metricService)
	httpserver := httputil.NewHttpServer(config, router.RouterSetup(), certificatesService)
	app := NewMain(
		config,
		emailService,
		grpcServer,
		httpserver,
		consulClient,
		serviceID,
	)

	return app, nil
}
