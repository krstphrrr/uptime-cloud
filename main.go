package main

import (
	"encoding/json"
	"fmt"
	// "io"
	"strings"
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

var alertState = sync.Map{} // Map to track alert state for each website

type AlertState struct {
    LastAlertTime    time.Time
    ConsecutiveFails int
}

type Config struct {
    Websites        []Website `json:"websites"`
    Email           Email     `json:"email"`
    FailureThreshold int      `json:"failure_threshold"`
    DebounceDuration float64  `json:"debounce_duration"` // Change to float64
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
	
	
	fmt.Printf("Using smtp host: %s\n", config.Email.SMTPHost)
	fmt.Printf("Using smtp port: %d\n", config.Email.SMTPPort)

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
			pollWebsite(site, config)
		}(website)
	}


	wg.Wait()
}

func loadConfig(filePath string) (*Config, error) {
    rawConfig, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    // Replace placeholders with environment variables
    processedConfig := os.ExpandEnv(string(rawConfig))

    if usernameFile := os.Getenv("SMTP_USERNAME_FILE"); usernameFile != "" {
		username, err := os.ReadFile(usernameFile)
		if err == nil {
			os.Setenv("SMTP_USERNAME", strings.TrimSpace(string(username)))
		} else {
			log.Printf("Failed to read SMTP_USERNAME_FILE: %v", err)
		}
	}
	
	if passwordFile := os.Getenv("SMTP_PASSWORD_FILE"); passwordFile != "" {
		password, err := os.ReadFile(passwordFile)
		if err == nil {
			os.Setenv("SMTP_PASSWORD", strings.TrimSpace(string(password)))
		} else {
			log.Printf("Failed to read SMTP_PASSWORD_FILE: %v", err)
		}
	}
	

    var config Config
    err = json.Unmarshal([]byte(processedConfig), &config)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    return &config, nil
}

func pollWebsite(site Website, config *Config) {
    failureThreshold := config.FailureThreshold
	debounceDuration := time.Duration(config.DebounceDuration * float64(time.Minute))

    for {
        start := time.Now()
        resp, err := http.Get(site.URL)
        responseTime := time.Since(start).Seconds()

        status := 0.0
        if err != nil {
            log.Printf("Website DOWN: %s (Error: %v)", site.URL, err)
            handleFailure(site, config.Email, failureThreshold, debounceDuration, "Connection error or timeout")
        } else {
            defer resp.Body.Close()

            if resp.StatusCode >= 500 {
                log.Printf("Website DOWN: %s (Server error: %d)", site.URL, resp.StatusCode)
                handleFailure(site, config.Email, failureThreshold, debounceDuration, fmt.Sprintf("Server error: %d", resp.StatusCode))
            } else if resp.StatusCode >= 400 {
                log.Printf("Website DOWN: %s (Client error: %d)", site.URL, resp.StatusCode)
                handleFailure(site, config.Email, failureThreshold, debounceDuration, fmt.Sprintf("Client error: %d", resp.StatusCode))
            } else {
                log.Printf("Website UP: %s (%d)", site.URL, resp.StatusCode)
                status = 1.0

                // Reset alert state on successful response
                alertState.Store(site.URL, &AlertState{
                    LastAlertTime:    time.Now(),
                    ConsecutiveFails: 0,
                })
            }
        }

        uptimeGauge.WithLabelValues(site.URL).Set(status)
        responseTimeHistogram.WithLabelValues(site.URL).Observe(responseTime)

        time.Sleep(time.Duration(site.PollInterval) * time.Second)
    }
}

func handleFailure(site Website, emailConfig Email, failureThreshold int, debounceDuration time.Duration, reason string) {
    state, _ := alertState.LoadOrStore(site.URL, &AlertState{})
    alert := state.(*AlertState)

    // Update consecutive failures
    alert.ConsecutiveFails++

    // Check if it's time to send an alert
    if alert.ConsecutiveFails >= failureThreshold && time.Since(alert.LastAlertTime) > debounceDuration {
        alert.LastAlertTime = time.Now()
        sendAlert(emailConfig, site.Custodians, site.URL, reason)
    }
}
// mail handler
func sendAlert(emailConfig Email, custodians []string, url string, reason string) {
    dialer := gomail.NewDialer(emailConfig.SMTPHost, emailConfig.SMTPPort, emailConfig.Username, emailConfig.Password)

    message := gomail.NewMessage()
    message.SetHeader("From", emailConfig.Username)

    recipients := validateEmails(custodians)
    if len(recipients) == 0 {
        log.Println("No valid recipients for the alert email.")
        return
    }

    message.SetHeader("To", recipients...)
    message.SetHeader("Subject", "Website Down Alert")
    message.SetBody("text/plain", fmt.Sprintf("The website %s is down. Reason: %s", url, reason))

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