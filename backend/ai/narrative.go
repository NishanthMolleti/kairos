package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// BuildNarrative queries all health metric tables for userID on date and returns
// a descriptive string suitable for embedding and RAG retrieval.
func BuildNarrative(db *sqlx.DB, userID uuid.UUID, date time.Time) (string, error) {
	dateStr := date.Format("January 2, 2006")
	dateFmt := date.Format("2006-01-02")
	parts := []string{fmt.Sprintf("%s:", dateStr)}

	// --- daily_sleep ---
	var sleepScore, efficiency, latency, restless *int
	var totalSleep, rem, deep, light, awake *int
	_ = db.QueryRow(
		`SELECT score, total_sleep_duration, efficiency, latency,
		        rem_sleep_duration, deep_sleep_duration, light_sleep_duration,
		        awake_time, restless_periods
		 FROM daily_sleep WHERE user_id=$1 AND date=$2`,
		userID, dateFmt,
	).Scan(&sleepScore, &totalSleep, &efficiency, &latency, &rem, &deep, &light, &awake, &restless)
	if sleepScore != nil {
		parts = append(parts, fmt.Sprintf("sleep score %d", *sleepScore))
	}
	if totalSleep != nil {
		h := *totalSleep / 3600
		m := (*totalSleep % 3600) / 60
		parts = append(parts, fmt.Sprintf("total sleep %dh%dm", h, m))
	}
	if efficiency != nil {
		parts = append(parts, fmt.Sprintf("sleep efficiency %d%%", *efficiency))
	}
	if latency != nil {
		parts = append(parts, fmt.Sprintf("sleep latency %dmin", *latency/60))
	}
	if rem != nil {
		parts = append(parts, fmt.Sprintf("REM %dmin", *rem/60))
	}
	if deep != nil {
		parts = append(parts, fmt.Sprintf("deep sleep %dmin", *deep/60))
	}
	if restless != nil {
		parts = append(parts, fmt.Sprintf("restless periods %d", *restless))
	}

	// --- daily_readiness ---
	var readScore, rhr, recoveryIndex, hrvBalance, activityBalance, sleepBalance *int
	var bodyTemp *float64
	_ = db.QueryRow(
		`SELECT score, hrv_balance, body_temperature, recovery_index,
		        resting_heart_rate, activity_balance, sleep_balance
		 FROM daily_readiness WHERE user_id=$1 AND date=$2`,
		userID, dateFmt,
	).Scan(&readScore, &hrvBalance, &bodyTemp, &recoveryIndex, &rhr, &activityBalance, &sleepBalance)
	if readScore != nil {
		parts = append(parts, fmt.Sprintf("readiness %d", *readScore))
	}
	if rhr != nil {
		parts = append(parts, fmt.Sprintf("resting HR %dbpm", *rhr))
	}
	if bodyTemp != nil {
		parts = append(parts, fmt.Sprintf("body temp deviation %.1f°C", *bodyTemp))
	}
	if recoveryIndex != nil {
		parts = append(parts, fmt.Sprintf("recovery index %d", *recoveryIndex))
	}

	// --- daily_activity ---
	var actScore, steps, calories, activeCalories *int
	var metMinutes *float64
	_ = db.QueryRow(
		`SELECT score, steps, calories, active_calories, met_minutes
		 FROM daily_activity WHERE user_id=$1 AND date=$2`,
		userID, dateFmt,
	).Scan(&actScore, &steps, &calories, &activeCalories, &metMinutes)
	if actScore != nil {
		parts = append(parts, fmt.Sprintf("activity score %d", *actScore))
	}
	if steps != nil {
		parts = append(parts, fmt.Sprintf("steps %d", *steps))
	}
	if calories != nil {
		parts = append(parts, fmt.Sprintf("total calories %d", *calories))
	}
	if activeCalories != nil {
		parts = append(parts, fmt.Sprintf("active calories %d", *activeCalories))
	}

	// --- daily_hrv ---
	var rmssd, bdi *float64
	_ = db.QueryRow(
		`SELECT rmssd, bdi FROM daily_hrv WHERE user_id=$1 AND date=$2`,
		userID, dateFmt,
	).Scan(&rmssd, &bdi)
	if rmssd != nil {
		parts = append(parts, fmt.Sprintf("HRV %.0fms", *rmssd))
	}
	if bdi != nil {
		parts = append(parts, fmt.Sprintf("HRV balance %.1f", *bdi))
	}

	// --- daily_spo2 ---
	var avgSpO2, minSpO2 *float64
	_ = db.QueryRow(
		`SELECT avg_spo2, min_spo2 FROM daily_spo2 WHERE user_id=$1 AND date=$2`,
		userID, dateFmt,
	).Scan(&avgSpO2, &minSpO2)
	if avgSpO2 != nil {
		parts = append(parts, fmt.Sprintf("avg SpO2 %.1f%%", *avgSpO2))
	}
	if minSpO2 != nil {
		parts = append(parts, fmt.Sprintf("min SpO2 %.1f%%", *minSpO2))
	}

	// --- daily_stress ---
	var stressHigh, recoveryHigh *int
	var daySummary *string
	_ = db.QueryRow(
		`SELECT stress_high, recovery_high, day_summary
		 FROM daily_stress WHERE user_id=$1 AND date=$2`,
		userID, dateFmt,
	).Scan(&stressHigh, &recoveryHigh, &daySummary)
	if stressHigh != nil {
		parts = append(parts, fmt.Sprintf("high-stress time %dmin", *stressHigh/60))
	}
	if recoveryHigh != nil {
		parts = append(parts, fmt.Sprintf("recovery time %dmin", *recoveryHigh/60))
	}
	if daySummary != nil {
		parts = append(parts, fmt.Sprintf("day summary: %s", *daySummary))
	}

	// --- workouts ---
	rows, err := db.Query(
		`SELECT activity, calories, distance,
		        EXTRACT(EPOCH FROM (end_datetime - start_datetime))::int AS duration_sec
		 FROM workouts WHERE user_id=$1 AND DATE(start_datetime)=$2`,
		userID, dateFmt,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var activity *string
			var wCal *int
			var wDist *float64
			var durSec *int
			if scanErr := rows.Scan(&activity, &wCal, &wDist, &durSec); scanErr == nil {
				wParts := []string{}
				if activity != nil {
					wParts = append(wParts, *activity)
				}
				if durSec != nil {
					wParts = append(wParts, fmt.Sprintf("%dmin", *durSec/60))
				}
				if wCal != nil {
					wParts = append(wParts, fmt.Sprintf("%d cal", *wCal))
				}
				if wDist != nil && *wDist > 0 {
					wParts = append(wParts, fmt.Sprintf("%.2fkm", *wDist/1000))
				}
				if len(wParts) > 0 {
					parts = append(parts, "workout: "+strings.Join(wParts, " "))
				}
			}
		}
	}

	if len(parts) == 1 {
		return "", fmt.Errorf("narrative: no data found for user %s on %s", userID, dateFmt)
	}
	return strings.Join(parts, ", "), nil
}
