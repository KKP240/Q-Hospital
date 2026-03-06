package main

import (
	"encoding/json"
	"log"

	"github.com/KKP240/Q-Hospital/circuitbreaker"
	"github.com/sony/gobreaker"
)

var queueBreaker *gobreaker.CircuitBreaker

func init() {
	queueBreaker = circuitbreaker.NewBreaker("queue-db")
}

func startConsumer() {

	q, err := rabbitCh.QueueDeclare(
		"queue-patient",
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Fatal("Queue declare failed:", err)
	}

	err = rabbitCh.QueueBind(
		q.Name,
		"appointment.confirmed",
		"hospital",
		false,
		nil,
	)

	if err != nil {
		log.Fatal("Queue bind failed:", err)
	}

	msgs, err := rabbitCh.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Fatal("Consume failed:", err)
	}

	go func() {

		for msg := range msgs {

			var event map[string]interface{}

			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Println("Invalid event:", err)
				msg.Nack(false, false)
				continue
			}

			queue := Queue{
				AppointmentID: uint(event["appointment_id"].(float64)),
				PatientID:     event["patient_id"].(string),
				DoctorID:      event["doctor_id"].(string),
				Status:        StatusWaiting,
			}

			// Circuit Breaker ครอบ DB
			_, err := queueBreaker.Execute(func() (interface{}, error) {
				if err := db.Create(&queue).Error; err != nil {
					return nil, err
				}
				return true, nil
			})

			if err != nil {

				log.Println("Queue creation failed:", err)

				if err == gobreaker.ErrOpenState {
					log.Println("Circuit breaker OPEN - skipping DB call")
				}

				msg.Nack(false, true)
				continue
			}

			log.Printf("Queue created %s", queue.QueueNumber)

			msg.Ack(false)
		}

	}()
}
