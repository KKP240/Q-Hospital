package main

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
	"gorm.io/gorm"
)

func initRabbitMQConsumer(ch *amqp.Channel, db *gorm.DB) {
	// รับข้อมูลจาก queue-patient
	err := ch.ExchangeDeclare("hospital", "topic", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	q, err := ch.QueueDeclare("payment-pill", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	err = ch.QueueBind(q.Name, "queue.done", "hospital", false, nil)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	go func() {
		msgs, _ := ch.Consume(q.Name, "", false, false, false, false, nil)
		for msg := range msgs {
			var event map[string]interface{}
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}

			queueID, ok := event["queue_id"].(float64)
			if !ok {
				log.Println("Invalid queue_id in message")
				continue
			}

			var count int64
			db.Model(&Payment{}).Where("queue_id = ?", uint(queueID)).Count(&count)
			if count > 0 {
				msg.Ack(false)
				continue
			}

			// สร้างใบเสร็จ
			payment := Payment{QueueID: uint(queueID), Amount: 500}
			if err := db.Create(&payment).Error; err != nil {
				log.Printf("Failed to create: %v", err)
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
			}
		}
	}()
}
