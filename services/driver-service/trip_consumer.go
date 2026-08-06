package main

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func NewTripEventConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripConsumer {

	return &tripConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}

func (t *tripConsumer) Listen() error {
	return t.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {

		var tripEvent contracts.AmqpMessage

		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return err
		}

		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("driver recieved message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return t.handleAndNotifyDrivers(ctx, payload)
		}

		log.Printf("unknown trip event: %+v", payload)
		return nil
	})
}

func (t *tripConsumer) handleAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {

	suitableIDs := t.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)

	log.Printf("found suitable drivers: %d", len(suitableIDs))

	if len(suitableIDs) == 0 {

		if err := t.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("failed to publish message to exchange: %v", err)
		}

		return nil
	}

	suitableDriverID := suitableIDs[0]

	marshaledEvent, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	//notify driver about potential trip
	if err := t.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshaledEvent,
	}); err != nil {
		log.Printf("failed to publish message to exchange: %v", err)
	}

	return nil
}
