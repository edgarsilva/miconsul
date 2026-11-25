import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:3200";
const EMAIL = __ENV.AUTH_EMAIL || "admin@seed.local";
const PASSWORD = __ENV.AUTH_PASSWORD || "Admin123!";
const VUS = Number(__ENV.VUS || 20);
const DURATION = __ENV.DURATION || "30s";

export const options = {
  noCookiesReset: true,
  scenarios: {
    mix: { executor: "constant-vus", vus: VUS, duration: DURATION },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
    checks: ["rate>0.99"],
  },
};

const reads = [
  { weight: 25, name: "dashboard", path: () => "/dashboard" },
  { weight: 25, name: "appointments", path: () => "/appointments/?timeframe=month" },
  { weight: 15, name: "clinic search", path: () => "/clinics/search?searchTerm=Seed" },
  { weight: 15, name: "patient search", path: () => "/patients/search?searchTerm=Alm" },
  { weight: 10, name: "revenue", path: () => "/revenue" },
];
const writeWeight = 10;

function signedInPage(res) {
  return res.status === 200 && !res.url.includes("/signin");
}

function signIn() {
  const res = http.post(`${BASE_URL}/signin`, { email: EMAIL, password: PASSWORD }, { redirects: 0, tags: { name: "signin" } });
  check(res, { "signed in": (r) => r.status === 303 && r.cookies.session_id !== undefined });
}

function uids(body, prefix) {
  const found = new Set();
  const re = new RegExp(`href="/${prefix}/([a-z0-9]{20,})"`, "g");
  for (const m of body.matchAll(re)) {
    found.add(m[1]);
  }
  return [...found];
}

export function setup() {
  signIn();
  const clinics = uids(http.get(`${BASE_URL}/clinics/`).body, "clinics");
  const patients = uids(http.get(`${BASE_URL}/patients/`).body, "patients");
  const appointments = uids(http.get(`${BASE_URL}/appointments/?timeframe=month`).body, "appointments");
  if (clinics.length === 0 || patients.length === 0) {
    throw new Error("the seeded user has no clinics or patients; run scripts/stack.sh up");
  }
  return { clinics, patients, appointments };
}

let signedIn = false;

export default function (data) {
  if (!signedIn) {
    signIn();
    signedIn = true;
  }

  const total = reads.reduce((sum, r) => sum + r.weight, writeWeight);
  let draw = Math.random() * total;
  for (const r of reads) {
    draw -= r.weight;
    if (draw < 0) {
      const res = http.get(`${BASE_URL}${r.path()}`, { tags: { name: r.name } });
      check(res, { [`${r.name} 200`]: signedInPage });
      return;
    }
  }

  write(data);
}

function pick(list) {
  return list[Math.floor(Math.random() * list.length)];
}

function bookedAt() {
  const day = 1 + Math.floor(Math.random() * 28);
  const hour = 8 + Math.floor(Math.random() * 10);
  const pad = (n) => String(n).padStart(2, "0");
  return `2027-03-${pad(day)}T${pad(hour)}:00`;
}

function write(data) {
  const form = { clinicId: pick(data.clinics), patientId: pick(data.patients), duration: "30", bookedAt: bookedAt(), price: "500" };
  if (data.appointments.length > 0 && Math.random() < 0.5) {
    const res = http.patch(`${BASE_URL}/appointments/${pick(data.appointments)}`, form, { tags: { name: "update appointment" } });
    check(res, { "update appointment 200": signedInPage });
    return;
  }
  const res = http.post(`${BASE_URL}/appointments`, form, { tags: { name: "create appointment" } });
  check(res, { "create appointment 200": signedInPage });
}
