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
	"syscall"

	"ride-sharing/services/trip-service/internal/infrastructure/grpc"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = ":9093"
)

func main() {
	inmemRepo := repository.NewInmemRepository()
	svc := service.NewTripService(inmemRepo)

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

	publisher := events.NewTripEventPublisher(rabbitMQ)

	driverConsumer := events.NewDriverConsumer(rabbitMQ, svc)
	go driverConsumer.Listen()

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
