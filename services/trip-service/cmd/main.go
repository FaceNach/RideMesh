package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	"ride-sharing/services/trip-service/internal/infrastructure/grpc"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = ":9093"
)

func main() {

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "trip-service",
		Enviroment:     env.GetString("ENVIROMENT", "development"),
		JaegerEndPoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer sh(ctx)
	defer cancel()

	inmemRepo := repository.NewInmemRepository()
	svc := service.NewTripService(inmemRepo)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	publisher := events.NewTripEventPublisher(rabbitmq)

	driverConsumer := events.NewDriverConsumer(rabbitmq, svc)
	go driverConsumer.Listen()

	paymentConsumer := events.NewPaymentConsumer(rabbitmq, svc)
	go paymentConsumer.Listen()

	grpcServer := grpcserver.NewServer()
	//TODO initialize our grpc handler implementation
	grpc.NewgRPCHandler(grpcServer, svc, publisher)
	log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())

	//grpcServerErrors := make(chan error, 1)

	go func() {
		//grpcServerErrors <- grpcServer.Serve(lis)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to server: %v", err)
			cancel()
		}
	}()

	//wait for the shutdown signal
	<-ctx.Done()
	log.Println("Shutting down server")
	grpcServer.GracefulStop()

}
