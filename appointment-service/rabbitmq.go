package main

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

var rabbitCh *amqp.Channel

// RabbitMQ Publisher
func publishEvent(ctx context.Context, routingKey string, a Appointment) error {
	event, err := json.Marshal(map[string]interface{}{
		"appointment_id": a.ID,
		"patient_id":     a.PatientID,
		"doctor_id":      a.DoctorID,
	})

	if err != nil {
		return err
	}

	return rabbitCh.PublishWithContext(
		ctx,
		"hospital",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        event,
		},
	)
}
