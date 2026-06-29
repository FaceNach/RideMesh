package http

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/types"
)

type HttpHandler struct {
	Service *service.Service
}

type previewTripRquest struct {
	UserID      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

func (s *HttpHandler) HandleTripPreview(w http.ResponseWriter, r *http.Request) {

	var reqBody previewTripRquest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {

		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	ctx := r.Context()

	fare := &domain.RideFareModel{
		UserID: "42",
	}

	trip, err := s.Service.CreateTrip(ctx, fare)
	if err != nil {
		log.Printf("%v", err)
	}

	if err := writeJSON(w, http.StatusOK, trip); err != nil {

		log.Printf("%v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
