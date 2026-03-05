package main

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

var rabbitCh *amqp.Channel

func publishEvent(routingKey string, payload interface{}) error {

	event, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return rabbitCh.Publish(
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
