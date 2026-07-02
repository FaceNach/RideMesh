package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	service domain.TripService
}

func NewgRPCHandler(server *grpc.Server, service domain.TripService) *gRPCHandler {
	handler := &gRPCHandler{
		service: service,
	}

	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {

	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	pickupCoord := &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}

	userID := req.GetUserID()

	destinationCoord := &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}

	t, err := h.service.GetRoute(ctx, pickupCoord, destinationCoord)
	if err != nil {
		log.Printf("error trying to get trip route: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(t)
	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, userID)
	if err != nil {
		log.Printf("error trying to generate trip fares: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to generate ride fares: %v", err)
	}

	faresToProto := domain.ToRideFaresProto(fares)

	return &pb.PreviewTripResponse{
		Route:     t.ToProto(),
		RideFares: faresToProto,
	}, nil
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateTrip not implemented")
}
