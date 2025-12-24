package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"email-scheduler/middleware"
	"email-scheduler/scheduler"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found.")
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	checkSchedule := os.Getenv("SCHEDULE_CRON")
	smtpMock := os.Getenv("SMTP_MOCK") == "true"
	apiToken := os.Getenv("API_TOKEN")

	fmt.Printf("SMTP Host: %s:%s\n", smtpHost, smtpPort)
	fmt.Printf("SMTP User: %s\n", smtpEmail)

	sched := scheduler.NewScheduler(scheduler.SmtpConfig{
		Host:     smtpHost,
		Port:     smtpPort,
		From:     smtpEmail,
		Password: smtpPass,
		MockMode: smtpMock,
	}, checkSchedule)

	sched.Start()

	auth := middleware.AuthMiddleware(apiToken)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	http.HandleFunc("/schedule", auth(func(w http.ResponseWriter, r *http.Request) {
		var email scheduler.EmailData
		if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		email.SendAt = sched.CalculateScheduleTime(time.Now())
		sched.AddEmailToQueue(email)

		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{
			"status":  "queued",
			"message": fmt.Sprintf("Email to %s scheduled for %s", email.To, email.SendAt.Format(time.RFC3339)),
			"send_at": email.SendAt.Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))

	log.Println("Server listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
