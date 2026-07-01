// scheduler/cron.go
package scheduler

import (
	"context"
	"log"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/NishanthMolleti/kairos/oura"
	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
)

func Start(db *sqlx.DB) *cron.Cron {
	c := cron.New()
	c.AddFunc("0 3 * * *", func() { runSync(db) })
	c.Start()
	log.Println("cron scheduler started (daily at 03:00 UTC)")
	return c
}

func runSync(db *sqlx.DB) {
	users, err := models.GetAllUsers(db)
	if err != nil {
		log.Printf("cron: get users failed: %v", err)
		return
	}
	log.Printf("cron: syncing %d users", len(users))
	for _, u := range users {
		if err := oura.SyncUser(context.Background(), db, u.ID, u.AccessToken); err != nil {
			log.Printf("cron: sync failed for user %s: %v", u.ID, err)
		} else {
			log.Printf("cron: synced user %s", u.ID)
		}
	}
}
