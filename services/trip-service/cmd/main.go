package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	h "ride-sharing/services/trip-service/internal/infrastructure/http"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"syscall"
	"time"
)

var (
	httpAddr = ":8083"
)

func main() {
	inmemRepo := repository.NewInmemRepository()
	svc := service.NewTripService(inmemRepo)

	mux := http.NewServeMux()

	handler := h.HttpHandler{Service: svc}

	mux.HandleFunc("POST /preview", handler.HandleTripPreview)

	server := &http.Server{Addr: httpAddr, Handler: mux}

	errChan := make(chan error, 1)

	go func() {
		log.Println("Starting Trip Service")
		errChan <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Printf("error starting the trip-service server: %v", err)
	case sig := <-shutdown:
		log.Printf("server is shutting down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("could not stop server gracefully:  %v", err)
			server.Close()
		}
	}

}
