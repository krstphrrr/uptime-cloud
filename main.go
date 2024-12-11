package main

import (
	"encoding/json"
	"fmt"
	"strings"

	// "io"
	// "io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"uptime_monitor/version"

	"github.com/go-gomail/gomail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Websites []Website `json:"websites"`
	Email    Email     `json:"email"`
}

type Website struct {
	URL          string   `json:"url"`
	PollInterval int      `json:"poll_interval"`
	Custodians   []string `json:"custodians"`
}

type Email struct {
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	uptimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "website_up",
			Help: "Website up status (1 = UP, 0 = DOWN)",
		},
		[]string{"url"},
	)

	responseTimeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "website_response_time_seconds",
			Help:    "Response time for websites in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"url"},
	)
)

func init() {
	prometheus.MustRegister(uptimeGauge)
	prometheus.MustRegister(responseTimeHistogram)
}

func main() {

	fmt.Printf("Uptime Monitor Version: %s\n", version.AppVersion)
	// load config
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// start prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Starting Prometheus metrics server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	var wg sync.WaitGroup
	for _, website := range config.Websites {
		wg.Add(1)
		go func(site Website) {
			defer wg.Done()
			pollWebsite(site, config.Email)
		}(website)
	}
	fmt.Printf("Using smtp host: %s\n", config.Email.SMTPHost)
	fmt.Printf("Using smtp port: %s\n", config.Email.SMTPPort)


	wg.Wait()
}

func loadConfig(filePath string) (*Config, error) {
	// Read the raw config file
	rawConfig, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Replace placeholders with environment variables
	processedConfig := os.ExpandEnv(string(rawConfig))

	// Unmarshal into the Config struct
	var config Config
	err = json.Unmarshal([]byte(processedConfig), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func pollWebsite(site Website, emailConfig Email) {
	for {
		start := time.Now()
		resp, err := http.Get(site.URL)
		responseTime := time.Since(start).Seconds()

		status := 0.0
		if err != nil || resp.StatusCode != 200 {
			log.Printf("Website DOWN: %s (%v)", site.URL, err)
			status = 0.0

			// Log and send email
			log.Printf("Custodians for %s: %v", site.URL, site.Custodians)
			sendAlert(emailConfig, site.Custodians, site.URL)
		} else {
			log.Printf("Website UP: %s (%d)", site.URL, resp.StatusCode)
			status = 1.0
		}

		uptimeGauge.WithLabelValues(site.URL).Set(status)
		responseTimeHistogram.WithLabelValues(site.URL).Observe(responseTime)

		time.Sleep(time.Duration(site.PollInterval) * time.Second)
	}
}

// mail handler
func sendAlert(emailConfig Email, custodians []string, url string) {
	dialer := gomail.NewDialer(emailConfig.SMTPHost, emailConfig.SMTPPort, emailConfig.Username, emailConfig.Password)

	message := gomail.NewMessage()
	message.SetHeader("From", emailConfig.Username)

	// Ensure recipients list is non-empty and valid
	recipients := validateEmails(custodians)
	if len(recipients) == 0 {
		log.Println("No valid recipients for the alert email.")
		return
	}

	message.SetHeader("To", recipients...) // Spread the recipients slice

	message.SetHeader("Subject", "Website Down Alert")
	message.SetBody("text/plain", fmt.Sprintf("The website %s is down.", url))

	// Send the email
	if err := dialer.DialAndSend(message); err != nil {
		log.Printf("Failed to send alert email: %v", err)
	}
}

//  email validation

func validateEmails(emails []string) []string {
	validEmails := []string{}
	for _, email := range emails {
		log.Printf("Processing email: %s", email)
		if email != "" && strings.Contains(email, "@") {
			validEmails = append(validEmails, email)
		} else {
			log.Printf("Invalid email address: %s", email)
		}
	}
	return validEmails
}