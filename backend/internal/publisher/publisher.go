package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPublisher struct {
	channel *amqp.Channel
}

func NewEventPublisher(ch *amqp.Channel) *EventPublisher {
	return &EventPublisher{channel: ch}
}

// SetupExchange memastikan Exchange dengan tipe Topic/Direct dideklarasikan sebagai Durable
func (p *EventPublisher) SetupExchange(exchangeName, exchangeType string) error {
	return p.channel.ExchangeDeclare(
		exchangeName, // nama exchange (misal: "order.events")
		exchangeType, // tipe exchange (misal: "topic" atau "direct")
		true,         // durable = true (Exchange bertahan meski broker restart)
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}

// PublishDomainEvent menerbitkan payload domain event ke RabbitMQ dengan jaminan Persistensi Penuh
func (p *EventPublisher) PublishDomainEvent(ctx context.Context, exchange, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal JSON payload: %w", err)
	}

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Menerbitkan pesan dengan DeliveryMode = amqp.Persistent (Nilai 2)
	err = p.channel.PublishWithContext(pubCtx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Wajib: Persistent delivery mode
			ContentType:  "application/json",
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("gagal menerbitkan event ke exchange %s: %w", exchange, err)
	}

	log.Printf("[PUBLISHER] Event berhasil dikirim ke Exchange: '%s' | RoutingKey: '%s'", exchange, routingKey)
	return nil
}