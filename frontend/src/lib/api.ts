import { getToken } from './auth';
import type {
  User, DailySleep, DailyReadiness, DailyActivity,
  DailyHRV, HeartRate, DailySpO2, DailyStress, Workout, ChatMessage,
} from './types';

const BASE = 'https://kairos.nimoclaw.dev';

function authHeaders(): HeadersInit {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...authHeaders(), ...init.headers },
  });
  if (!res.ok) throw new Error(`API error ${res.status}: ${await res.text()}`);
  return res.json() as Promise<T>;
}

type DateRange = { from: string; to: string };

function dateParams({ from, to }: DateRange): string {
  return `?from=${from}&to=${to}`;
}

export const getUser = (): Promise<User> => apiFetch('/api/user');
export const postSync = (): Promise<{ message: string }> =>
  apiFetch('/api/sync', { method: 'POST' });

export const getSleep = (r: DateRange): Promise<DailySleep[]> =>
  apiFetch(`/api/metrics/sleep${dateParams(r)}`);
export const getReadiness = (r: DateRange): Promise<DailyReadiness[]> =>
  apiFetch(`/api/metrics/readiness${dateParams(r)}`);
export const getActivity = (r: DateRange): Promise<DailyActivity[]> =>
  apiFetch(`/api/metrics/activity${dateParams(r)}`);
export const getHRV = (r: DateRange): Promise<DailyHRV[]> =>
  apiFetch(`/api/metrics/hrv${dateParams(r)}`);
export const getHeartRate = (r: DateRange): Promise<HeartRate[]> =>
  apiFetch(`/api/metrics/heartrate${dateParams(r)}`);
export const getSpO2 = (r: DateRange): Promise<DailySpO2[]> =>
  apiFetch(`/api/metrics/spo2${dateParams(r)}`);
export const getStress = (r: DateRange): Promise<DailyStress[]> =>
  apiFetch(`/api/metrics/stress${dateParams(r)}`);
export const getWorkouts = (r: DateRange): Promise<Workout[]> =>
  apiFetch(`/api/metrics/workouts${dateParams(r)}`);

export const createChatSession = (): Promise<{ id: string }> =>
  apiFetch('/api/chat/sessions', { method: 'POST', body: JSON.stringify({ title: 'New conversation' }) });
export const getChatMessages = (sessionId: string): Promise<ChatMessage[]> =>
  apiFetch(`/api/chat/sessions/${sessionId}/messages`);

// NOTE: backend expects { question: "..." } — NOT { message: "..." }
export function askStream(sessionId: string, question: string): Promise<Response> {
  return fetch(`${BASE}/api/chat/sessions/${sessionId}/ask`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ question }),
  });
}
