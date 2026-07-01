// oura/sync.go
package oura

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/NishanthMolleti/kairos/ai"
	"github.com/NishanthMolleti/kairos/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ouraSleep struct {
	Day                string `json:"day"`
	Score              *int   `json:"score"`
	TotalSleepDuration *int   `json:"total_sleep_duration"`
	Efficiency         *int   `json:"efficiency"`
	Latency            *int   `json:"latency"`
	RemSleepDuration   *int   `json:"rem_sleep_duration"`
	DeepSleepDuration  *int   `json:"deep_sleep_duration"`
	LightSleepDuration *int   `json:"light_sleep_duration"`
	AwakeTime          *int   `json:"awake_time"`
	RestlessPeriods    *int   `json:"restless_periods"`
}

type ouraReadiness struct {
	Day              string   `json:"day"`
	Score            *int     `json:"score"`
	HRVBalance       *int     `json:"hrv_balance"`
	BodyTemperature  *float64 `json:"body_temperature"`
	RecoveryIndex    *int     `json:"recovery_index"`
	RestingHeartRate *int     `json:"resting_heart_rate"`
	ActivityBalance  *int     `json:"activity_balance"`
	SleepBalance     *int     `json:"sleep_balance"`
}

type ouraActivity struct {
	Day                string   `json:"day"`
	Score              *int     `json:"score"`
	Steps              *int     `json:"steps"`
	TotalCalories      *int     `json:"total_calories"`
	ActiveCalories     *int     `json:"active_calories"`
	METMinutes         *float64 `json:"met_minutes_custom"`
	SedentaryTime      *int     `json:"sedentary_time"`
	LowActivityTime    *int     `json:"low_activity_time"`
	MediumActivityTime *int     `json:"medium_activity_time"`
	HighActivityTime   *int     `json:"high_activity_time"`
}

type ouraHRV struct {
	Day   string   `json:"day"`
	RMSSD *float64 `json:"rmssd"`
}

type ouraHeartRate struct {
	Timestamp string `json:"timestamp"`
	BPM       int    `json:"bpm"`
	Source    string `json:"source"`
}

type ouraSpO2 struct {
	Day         string   `json:"day"`
	SPO2Average *float64 `json:"spo2_percentage"`
}

type ouraStress struct {
	Day          string  `json:"day"`
	StressHigh   *int    `json:"stress_high"`
	RecoveryHigh *int    `json:"recovery_high"`
	DaySummary   *string `json:"day_summary"`
}

type ouraWorkout struct {
	StartDatetime string   `json:"start_datetime"`
	EndDatetime   string   `json:"end_datetime"`
	Activity      string   `json:"activity"`
	Calories      *int     `json:"calories"`
	Distance      *float64 `json:"distance"`
}

func dateRange() (string, string) {
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	return from, to
}

func SyncUser(ctx context.Context, db *sqlx.DB, userID uuid.UUID, accessToken string, hfAPIKey string) error {
	client := NewClient(accessToken)
	from, to := dateRange()
	params := url.Values{"start_date": {from}, "end_date": {to}}
	var errs []error

	// Sleep
	sleepData, err := get[ouraSleep](ctx, client, "/daily_sleep", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("sleep: %w", err))
	} else {
		for _, s := range sleepData {
			date, _ := time.Parse("2006-01-02", s.Day)
			models.UpsertDailySleep(db, &models.DailySleep{
				UserID: userID, Date: date,
				Score: s.Score, TotalSleepDuration: s.TotalSleepDuration,
				Efficiency: s.Efficiency, Latency: s.Latency,
				REMSleepDuration: s.RemSleepDuration, DeepSleepDuration: s.DeepSleepDuration,
				LightSleepDuration: s.LightSleepDuration, AwakeTime: s.AwakeTime,
				RestlessPeriods: s.RestlessPeriods,
			})
		}
	}

	// Readiness
	readinessData, err := get[ouraReadiness](ctx, client, "/daily_readiness", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("readiness: %w", err))
	} else {
		for _, r := range readinessData {
			date, _ := time.Parse("2006-01-02", r.Day)
			models.UpsertDailyReadiness(db, &models.DailyReadiness{
				UserID: userID, Date: date,
				Score: r.Score, HRVBalance: r.HRVBalance,
				BodyTemperature: r.BodyTemperature, RecoveryIndex: r.RecoveryIndex,
				RestingHeartRate: r.RestingHeartRate, ActivityBalance: r.ActivityBalance,
				SleepBalance: r.SleepBalance,
			})
		}
	}

	// Activity
	activityData, err := get[ouraActivity](ctx, client, "/daily_activity", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("activity: %w", err))
	} else {
		for _, a := range activityData {
			date, _ := time.Parse("2006-01-02", a.Day)
			models.UpsertDailyActivity(db, &models.DailyActivity{
				UserID: userID, Date: date,
				Score: a.Score, Steps: a.Steps,
				Calories: a.TotalCalories, ActiveCalories: a.ActiveCalories,
				METMinutes: a.METMinutes, SedentaryTime: a.SedentaryTime,
				LowActivity: a.LowActivityTime, MediumActivity: a.MediumActivityTime,
				HighActivity: a.HighActivityTime,
			})
		}
	}

	// HRV
	hrvData, err := get[ouraHRV](ctx, client, "/daily_hrv", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("hrv: %w", err))
	} else {
		for _, h := range hrvData {
			date, _ := time.Parse("2006-01-02", h.Day)
			models.UpsertDailyHRV(db, &models.DailyHRV{
				UserID: userID, Date: date, RMSSD: h.RMSSD,
			})
		}
	}

	// Heart Rate
	hrParams := url.Values{
		"start_datetime": {from + "T00:00:00Z"},
		"end_datetime":   {to + "T23:59:59Z"},
	}
	hrData, err := get[ouraHeartRate](ctx, client, "/heartrate", hrParams)
	if err != nil {
		errs = append(errs, fmt.Errorf("heartrate: %w", err))
	} else {
		hrRows := make([]models.HeartRate, 0, len(hrData))
		for _, h := range hrData {
			ts, _ := time.Parse(time.RFC3339, h.Timestamp)
			hrRows = append(hrRows, models.HeartRate{
				UserID: userID, Timestamp: ts, BPM: h.BPM, Source: h.Source,
			})
		}
		if err := models.BulkUpsertHeartRate(db, hrRows); err != nil {
			errs = append(errs, fmt.Errorf("heartrate upsert: %w", err))
		}
	}

	// SpO2
	spo2Data, err := get[ouraSpO2](ctx, client, "/daily_spo2", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("spo2: %w", err))
	} else {
		for _, s := range spo2Data {
			date, _ := time.Parse("2006-01-02", s.Day)
			models.UpsertDailySpO2(db, &models.DailySpO2{
				UserID: userID, Date: date, AvgSpO2: s.SPO2Average,
			})
		}
	}

	// Stress
	stressData, err := get[ouraStress](ctx, client, "/daily_stress", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("stress: %w", err))
	} else {
		for _, s := range stressData {
			date, _ := time.Parse("2006-01-02", s.Day)
			models.UpsertDailyStress(db, &models.DailyStress{
				UserID: userID, Date: date,
				StressHigh: s.StressHigh, RecoveryHigh: s.RecoveryHigh,
				DaySummary: s.DaySummary,
			})
		}
	}

	// Workouts
	workoutData, err := get[ouraWorkout](ctx, client, "/workout", params)
	if err != nil {
		errs = append(errs, fmt.Errorf("workouts: %w", err))
	} else {
		for _, w := range workoutData {
			start, _ := time.Parse(time.RFC3339, w.StartDatetime)
			var end *time.Time
			if w.EndDatetime != "" {
				t, _ := time.Parse(time.RFC3339, w.EndDatetime)
				end = &t
			}
			activity := w.Activity
			models.UpsertWorkout(db, &models.Workout{
				UserID: userID, StartDatetime: start, EndDatetime: end,
				Activity: &activity, Calories: w.Calories, Distance: w.Distance,
			})
		}
	}

	if len(errs) > 0 {
		log.Printf("sync user %s partial errors: %v", userID, errs)
	}

	// Generate narrative for today after sync completes.
	if err := ai.GenerateAndStoreNarrative(db, userID, time.Now(), hfAPIKey); err != nil {
		log.Printf("narrative generation failed for user %s: %v", userID, err)
	}

	return models.UpdateLastSync(db, userID)
}
