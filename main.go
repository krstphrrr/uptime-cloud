package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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
	// load config
	config := loadConfig("config.json")

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

	wg.Wait()
}

func loadConfig(filename string) Config {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return config
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
			// send mail
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
func sendAlert(emailConfig Email, recipients []string, url string) {
	dialer := gomail.NewDialer(emailConfig.SMTPHost, emailConfig.SMTPPort, emailConfig.Username, emailConfig.Password)

	message := gomail.NewMessage()
	message.SetHeader("From", emailConfig.Username)
	message.SetHeader("To", recipients...)
	message.SetHeader("Subject", "Website Down Alert")
	message.SetBody("text/plain", fmt.Sprintf("The website %s is down.", url))

	if err := dialer.DialAndSend(message); err != nil {
		log.Printf("Failed to send alert email: %v", err)
	}
}
