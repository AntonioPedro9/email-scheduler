package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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

	fmt.Println("=== Servidor de agendamento de emails ===")
	fmt.Println("Modo: Agendamento aleatório (07:00 - 21:00 BRT)")
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

	http.HandleFunc("/schedule", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var email scheduler.EmailData
		if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if email.To == "" {
			http.Error(w, "Field 'to' is required", http.StatusBadRequest)
			return
		}

		email.SendAt = sched.CalculateScheduleTime(time.Now())
		sched.AddEmailToQueue(email)
		log.Printf("Received request to schedule email to: %s (%s). Scheduled for: %s", email.To, email.Name, email.SendAt)

		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{
			"status":  "queued",
			"message": fmt.Sprintf("Email to %s scheduled for %s", email.To, email.SendAt.Format(time.RFC3339)),
			"send_at": email.SendAt.Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	})

	log.Println("Server listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
