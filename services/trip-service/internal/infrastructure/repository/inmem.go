package repository

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
)

type inmemRepository struct {
	trips     map[string]*domain.TripModel
	rideFares map[string]*domain.RideFareModel
}

func NewInmemRepository() *inmemRepository {
	return &inmemRepository{
		trips:     make(map[string]*domain.TripModel),
		rideFares: make(map[string]*domain.RideFareModel),
	}
}

func (r *inmemRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	r.trips[trip.ID.Hex()] = trip

	return trip, nil
}

func (r *inmemRepository) SaveRideFare(ctx context.Context, fare *domain.RideFareModel) error {
	r.rideFares[fare.ID.Hex()] = fare

	return nil
}

func (r *inmemRepository) GetRideFareById(ctx context.Context, id string) (*domain.RideFareModel, error) {
	fare, exist := r.rideFares[id]
	if !exist {
		return nil, fmt.Errorf("fare with ID %s does not exist", id)
	}

	return fare, nil
}
