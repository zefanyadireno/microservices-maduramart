package main

import (
	"encoding/json"
	"fmt"
)

func inspectPayload() {
	// 1. Representasi data objek untuk REST (JSON)
	restPayload := map[string]interface{}{
		"item": map[string]interface{}{
			"product_id":        "00000001-0000-0000-0000-000000000001",
			"quantity_on_hand":  100,
			"quantity_reserved": 5,
			"warehouse_id":      "w0000001-0000-0000-0000-000000000001",
		},
	}

	jsonBytes, err := json.Marshal(restPayload)
	if err != nil {
		panic(err)
	}

	// Simulasi ukuran biner custom untuk payload gRPC (misal disederhanakan tanpa file proto eksternal)
	// Struktur biner ringkas ala Protobuf (tag-length-value)
	protoBytes := []byte{0x0a, 0x3d, 0x0a, 0x24, '0', '0', '0', '0', '0', '0', '0', '1', '-', '0', '0', '0', '0', '-', '0', '0', '0', '0', '-', '0', '0', '0', '0', '-', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '1', 0x10, 0x64, 0x18, 0x05}

	// 3. Cetak hasil perbandingan ukuran byte ke terminal
	fmt.Printf("=== ANALISIS UKURAN PAYLOAD (WIRE-SIZE) ===\n")
	fmt.Printf("Ukuran Payload JSON (HTTP/1.1): %d Bytes\n", len(jsonBytes))
	fmt.Printf("Ukuran Payload gRPC (Protobuf): %d Bytes\n", len(protoBytes))

	diff := len(jsonBytes) - len(protoBytes)
	percentage := (float64(diff) / float64(len(jsonBytes))) * 100
	fmt.Printf("Selisih Penghematan Bandwidth: %d Bytes (Lebih ringkas %.2f%%)\n", diff, percentage)
}
