package events

import "time"

// OrderPlacedEvent adalah skema event saat pesanan baru berhasil dibuat
type OrderPlacedEvent struct {
	EventID   string          `json:"event_id"`   // UUID Unik per event
	EventType string          `json:"event_type"` // "OrderPlaced"
	Timestamp string          `json:"timestamp"`  // ISO-8601 Format
	Version   string          `json:"version"`    // "1.0"
	Producer  string          `json:"producer"`   // "order-service"
	Data      OrderPlacedData `json:"data"`
}

type OrderPlacedData struct {
	OrderID    string      `json:"order_id"`
	CustomerID string      `json:"customer_id"`
	TotalPrice float64     `json:"total_price"`
	Items      []OrderItem `json:"items"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// NewOrderPlacedEvent adalah helper untuk membuat instance event ber-metadata
func NewOrderPlacedEvent(eventID, orderID, customerID string, totalPrice float64, items []OrderItem) OrderPlacedEvent {
	return OrderPlacedEvent{
		EventID:   eventID,
		EventType: "OrderPlaced",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0",
		Producer:  "order-service",
		Data: OrderPlacedData{
			OrderID:    orderID,
			CustomerID: customerID,
			TotalPrice: totalPrice,
			Items:      items,
		},
	}
}