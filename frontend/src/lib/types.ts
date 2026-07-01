export interface DailySleep {
  id: string;
  date: string;
  score: number | null;
  total_sleep_duration: number | null;
  efficiency: number | null;
  latency: number | null;
  rem_sleep_duration: number | null;
  deep_sleep_duration: number | null;
  light_sleep_duration: number | null;
  awake_time: number | null;
  restless_periods: number | null;
}

export interface DailyReadiness {
  id: string;
  date: string;
  score: number | null;
  hrv_balance: number | null;
  body_temperature: number | null;
  recovery_index: number | null;
  resting_heart_rate: number | null;
  activity_balance: number | null;
  sleep_balance: number | null;
}

export interface DailyActivity {
  id: string;
  date: string;
  score: number | null;
  steps: number | null;
  calories: number | null;
  active_calories: number | null;
  met_minutes: number | null;
  sedentary_time: number | null;
  low_activity: number | null;
  medium_activity: number | null;
  high_activity: number | null;
}

export interface DailyHRV {
  id: string;
  date: string;
  rmssd: number | null;
  bdi: number | null;
}

export interface HeartRate {
  id: string;
  timestamp: string;
  bpm: number;
  source: string;
}

export interface DailySpO2 {
  id: string;
  date: string;
  avg_spo2: number | null;
  min_spo2: number | null;
}

export interface DailyStress {
  id: string;
  date: string;
  stress_high: number | null;
  recovery_high: number | null;
  day_summary: string | null;
}

export interface Workout {
  id: string;
  start_datetime: string;
  end_datetime: string | null;
  activity: string | null;
  calories: number | null;
  distance: number | null;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  sql_used: string | null;
  created_at: string;
}

export interface User {
  id: string;
  email: string;
  last_sync: string | null;
}
