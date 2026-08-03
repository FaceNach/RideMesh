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

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = ":9092"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
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
	grpcServer := grpcserver.NewServer()
	NewGrpcHandler(grpcServer, service)

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
