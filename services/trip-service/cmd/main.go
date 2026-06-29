package main

import (
	"log"
	"net/http"
	h "ride-sharing/services/trip-service/internal/infrastructure/http"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
)

var (
	httpAddr = ":8083"
)

func main() {
	inmemRepo := repository.NewInmemRepository()
	svc := service.NewTripService(inmemRepo)

	//keep the prog running

	log.Println("Starting API Gateway")

	mux := http.NewServeMux()

	handler := h.HttpHandler{Service: svc}

	mux.HandleFunc("POST /preview", handler.HandleTripPreview)

	server := &http.Server{Addr: httpAddr, Handler: mux}

	if err := server.ListenAndServe(); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}
