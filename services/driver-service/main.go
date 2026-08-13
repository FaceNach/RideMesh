package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	"syscall"

	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = ":9092"
)

func main() {

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
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
	rabbitMQ, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitMQ.Close()

	log.Println("Starting RabbitMQ connection")

	service := NewService()
	consumer := NewTripEventConsumer(rabbitMQ, service)

	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	NewGrpcHandler(grpcServer, service, consumer)

	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("failed to listen to the message: %v", err)
		}

	}()

	//TODO initialize our grpc handler implementation
	log.Printf("Starting gRPC server Driver service on port %s", lis.Addr().String())

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
