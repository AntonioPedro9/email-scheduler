package scheduler

import (
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type EmailData struct {
	To      string    `json:"to"`
	Name    string    `json:"name"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	SendAt  time.Time `json:"send_at"`
}

type SmtpConfig struct {
	Host     string
	Port     string
	From     string
	Password string
	MockMode bool
}

type Scheduler struct {
	cron       *cron.Cron
	queue      []EmailData
	mu         sync.Mutex // protects queue
	smtpConfig SmtpConfig
	location   *time.Location
}

func NewScheduler(config SmtpConfig, schedule string) *Scheduler {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Printf("Warning: Could not load Brazil location, using Local: %v", err)
		location = time.Local
	}

	s := &Scheduler{
		cron:       cron.New(),
		queue:      make([]EmailData, 0),
		smtpConfig: config,
		location:   location,
	}

	_, err = s.cron.AddFunc(schedule, s.processQueue)
	if err != nil {
		log.Fatalf("Error scheduling cron job: %v", err)
	}

	return s
}

func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("Scheduler started (checking queue every minute)")
}

func (s *Scheduler) CalculateScheduleTime(now time.Time) time.Time {
	nowInLocation := now.In(s.location)
	startHour := 7
	endHour := 21
	windowMinutes := (endHour - startHour) * 60 // (21 - 7) * 60 = 840 minutes
	randomOffset := rand.Intn(windowMinutes)
	year, month, day := nowInLocation.Date()
	todayStart := time.Date(year, month, day, startHour, 0, 0, 0, s.location)
	targetTime := todayStart.Add(time.Duration(randomOffset) * time.Minute)

	if targetTime.Before(nowInLocation) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	return targetTime
}

func (s *Scheduler) AddEmailToQueue(email EmailData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, email)
	log.Printf("Email scheduled for %s to %s (%s)", email.SendAt.Format(time.RFC3339), email.To, email.Name)
}

func (s *Scheduler) processQueue() {
	s.mu.Lock()
	if len(s.queue) == 0 {
		s.mu.Unlock()
		return
	}

	now := time.Now().In(s.location)
	var due []EmailData
	var remaining []EmailData

	for _, email := range s.queue {
		if now.After(email.SendAt) || now.Equal(email.SendAt) {
			due = append(due, email)
		} else {
			remaining = append(remaining, email)
		}
	}

	s.queue = remaining
	s.mu.Unlock()

	if len(due) > 0 {
		log.Printf("Processing %d due emails...", len(due))
		s.sendBatch(due)
	}
}

func (s *Scheduler) sendBatch(emails []EmailData) {
	auth := smtp.PlainAuth("", s.smtpConfig.From, s.smtpConfig.Password, s.smtpConfig.Host)
	addr := s.smtpConfig.Host + ":" + s.smtpConfig.Port

	for _, email := range emails {
		msg := []byte(fmt.Sprintf("To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s\r\n", email.To, email.Subject, email.Body))

		log.Printf("Sending email to %s...", email.To)

		if s.smtpConfig.Password == "" || s.smtpConfig.MockMode {
			log.Println("[MOCK] SendMail Success")
		} else {
			err := smtp.SendMail(addr, auth, s.smtpConfig.From, []string{email.To}, msg)
			if err != nil {
				log.Printf("ERROR sending to %s: %v", email.To, err)
			} else {
				log.Printf("SUCCESS sent to %s", email.To)
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}
