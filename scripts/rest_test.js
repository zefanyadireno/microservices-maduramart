import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 20 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  // Menggunakan rute publik /health yang sudah kita buat
  const res = http.get('http://localhost:3000/health');

  check(res, {
    'status REST adalah 200': (r) => r.status === 200,
  });

  sleep(1);
}