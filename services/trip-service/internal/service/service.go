package service

import (
	"context"
	"ride-sharing/services/trip-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	repo domain.TripRepository
}

func NewTripService(repo domain.TripRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	trip := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
	}

	out, _ := s.repo.CreateTrip(ctx, trip)

	return out, nil

}
