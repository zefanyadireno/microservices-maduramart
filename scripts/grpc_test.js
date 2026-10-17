import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';

const client = new grpc.Client();
// Memuat file proto untuk definisi layanan gRPC
client.load(['../proto'], 'inventory/v1/inventory.proto');

export const options = {
  stages: [
    { duration: '10s', target: 20 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  // Koneksi ke server gRPC lokal
  client.connect('localhost:50051', {
    plaintext: true,
  });

  const data = { product_id: '00000001-0000-0000-0000-000000000001' };
  
  // Memanggil remote procedure call (RPC)
  const response = client.invoke('inventory.v1.InventoryService/GetStockItem', data);

  check(response, {
    'status gRPC adalah OK': (r) => r && r.status === grpc.StatusOK,
  });

  client.close();
  sleep(1);
}