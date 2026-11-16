// 6.2: carga contra la instancia local. GALASH_TOKEN es una sesión sintética efímera (no se guarda).
import http from 'k6/http';
import { check } from 'k6';
const api = __ENV.GALASH_API_URL || 'http://localhost:18080';
const auth = { headers: { Authorization: `Bearer ${__ENV.GALASH_TOKEN}` }, tags: { name: 'GET /users/me' } };
export const options = {
  scenarios: { lectura: { executor: 'constant-vus', vus: 50, duration: '60s' } },
  thresholds: { 'http_req_duration{name:GET /seedbeds}': ['p(95)<300'], 'http_req_duration{name:GET /events}': ['p(95)<300'],
    'http_req_duration{name:GET /projects}': ['p(95)<300'], 'http_req_duration{name:GET /users/me}': ['p(95)<300'], http_req_failed: ['rate<0.01'] },
  summaryTrendStats: ['avg', 'p(50)', 'p(95)', 'p(99)', 'max'],
};
export default function () {
  for (const [path, params] of [['/seedbeds', { tags: { name: 'GET /seedbeds' } }], ['/events', { tags: { name: 'GET /events' } }], ['/projects', { tags: { name: 'GET /projects' } }], ['/users/me', auth]]) {
    check(http.get(api + path, params), { '200': r => r.status === 200 });
  }
}
