package consumer

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/nusantara-supermart/backend/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EventConsumer struct {
	channel *amqp.Channel
	db      *sql.DB
}

func NewEventConsumer(ch *amqp.Channel, db *sql.DB) *EventConsumer {
	return &EventConsumer{channel: ch, db: db}
}

func (c *EventConsumer) StartConsuming(queueName string) {
	// 1. QoS: Batasi consumer ambil maksimal 10 pesan sekaligus (mencegah overload memory)
	err := c.channel.Qos(10, 0, false)
	if err != nil {
		log.Fatalf("Gagal menyetel QoS: %v", err)
	}

	// 2. auto_ack = false: Wajib Manual ACK untuk memastikan keandalan
	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer tag
		false, // auto-ack = false
		false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Gagal mendaftarkan consumer: %v", err)
	}

	log.Printf("[CONSUMER] Menunggu event di queue: %s", queueName)

	go func() {
		for d := range msgs {
			var evt events.OrderPlacedEvent
			err := json.Unmarshal(d.Body, &evt)

			// 3. Poison Pill: Tolak pesan jika format JSON rusak (langsung ke Dead-Letter Queue)
			if err != nil {
				log.Printf("[ERROR] Poison pill (JSON tidak valid): %v", err)
				d.Reject(false) // requeue = false
				continue
			}

			// 4. Proses Idempotency: Eksekusi transaksi bisnis dengan aman
			err = c.processMessageIdempotent(evt)
			if err != nil {
				log.Printf("[ERROR] Gagal memproses event ID %s: %v", evt.EventID, err)
				d.Nack(false, false) // requeue = false -> kirim ke DLQ
				continue
			}

			// 5. Sukses: Kirim ACK ke RabbitMQ agar pesan dihapus
			d.Ack(false)
			log.Printf("[SUCCESS] Event Order %s (ID: %s) berhasil diproses", evt.Data.OrderID, evt.EventID)
		}
	}()
}

func (c *EventConsumer) processMessageIdempotent(evt events.OrderPlacedEvent) error {
	// Cek DB: Apakah event ini sudah pernah diproses?
	var exists bool
	queryCheck := "SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = ?)"
	err := c.db.QueryRow(queryCheck, evt.EventID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("[IDEMPOTENT] Event %s sudah pernah diproses. Melewati...", evt.EventID)
		return nil
	}

	// Mulai Transaksi Database
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// --- LOGIKA BISNIS: Memotong stok inventory berdasarkan Items ---
	log.Printf("Memproses pemotongan stok untuk Order: %s", evt.Data.OrderID)
	for _, item := range evt.Data.Items {
		log.Printf("- Mengurangi stok ProductID %s sebanyak %d pcs", item.ProductID, item.Quantity)
		// Contoh Eksekusi SQL nyata nanti:
		// tx.Exec("UPDATE products SET stock = stock - ? WHERE id = ?", item.Quantity, item.ProductID)
	}

	// Catat event_id ke tabel ledger idempotensi
	_, err = tx.Exec("INSERT INTO processed_events (event_id, event_type) VALUES (?, ?)", evt.EventID, evt.EventType)
	if err != nil {
		return err
	}

	// Commit transaksi
	return tx.Commit()
}
