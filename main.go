package main

import (
	"encoding/json"
	"fmt"
	"io"

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
    ConsecutiveSuccesses int // T\track consecutive successes
    AlertSent        bool   // whether a "website down" alert has been sent
}

type Config struct {
    Websites        []Website `json:"websites"`
    Email           Email     `json:"email"`
    FailureThreshold int      `json:"failure_threshold"`
	SuccessThreshold int      `json:"success_threshold"`
    // DebounceDuration float64  `json:"debounce_duration"` // Change to float64
}

type Website struct {
	URL          string   `json:"url"`
	PollInterval int      `json:"poll_interval"`
	Custodians   []string `json:"custodians"`
	SMSCustodians []string `json:"smsCustodians,omitempty"` // Optional field for SMS recipients
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
	config, err := loadConfig("config.development.json")
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
    // Load secrets first
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

    
    rawConfig, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    processedConfig := os.ExpandEnv(string(rawConfig))
    var config Config
    err = json.Unmarshal([]byte(processedConfig), &config)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    return &config, nil
}


func pollWebsite(site Website, config *Config) {
    failureThreshold := config.FailureThreshold
    successThreshold := config.SuccessThreshold
    client := &http.Client{
        Timeout: 10 * time.Second,
        Transport: &http.Transport{
            DisableKeepAlives: false, // Testing
            MaxIdleConns:        50,
            MaxIdleConnsPerHost: 2,
            IdleConnTimeout:     90 * time.Second, 
        },
    }

    for {
        start := time.Now()
        resp, err := client.Get(site.URL)
        if err != nil && strings.Contains(err.Error(), "EOF") {
            log.Printf("Retrying due to EOF error for %s", site.URL)
            resp, err = client.Get(site.URL)
        }
        responseTime := time.Since(start).Seconds()

        status := 0.0

        if err != nil {
            // Handle connection errors
            if err == io.EOF {
                log.Printf("EOF error from AWS deployment: %s (Connection closed early)", site.URL)
            } else {
                log.Printf("Website DOWN from AWS deployment: %s (Connection error: %v)", site.URL, err)
            }
            handleFailure(site, config.Email, failureThreshold, "Connection error or timeout")
        } else {
            if resp != nil {
                resp.Body.Close()
            }

            if resp.StatusCode >= 500 {
                // Handle server errors
                log.Printf("Website DOWN from AWS deployment: %s (Server error: %d)", site.URL, resp.StatusCode)
                handleFailure(site, config.Email, failureThreshold, fmt.Sprintf("Server error: %d", resp.StatusCode))
            } else if resp.StatusCode >= 400 {
                // Handle client errors
                log.Printf("Website DOWN from AWS deployment: %s (Client error: %d)", site.URL, resp.StatusCode)
                handleFailure(site, config.Email, failureThreshold, fmt.Sprintf("Client error: %d", resp.StatusCode))
            } else {
                // Handle success
                log.Printf("Website UP from AWS deployment: %s (Status code: %d)", site.URL, resp.StatusCode)
                handleSuccess(site, config.Email, successThreshold)
                status = 1.0
            }
        }

        // Update Prometheus metrics
        uptimeGauge.WithLabelValues(site.URL).Set(status)
        responseTimeHistogram.WithLabelValues(site.URL).Observe(responseTime)

        // Wait for the next poll
        time.Sleep(time.Duration(site.PollInterval) * time.Second)
    }
}




func handleFailure(site Website, emailConfig Email, failureThreshold int, reason string) {
    state, _ := alertState.LoadOrStore(site.URL, &AlertState{})
    alert := state.(*AlertState)

    alert.ConsecutiveFails++
    alert.ConsecutiveSuccesses = 0 // Reset successes on failure

    // Only send an alert if the failure threshold is met and no alert has been sent yet
    if alert.ConsecutiveFails >= failureThreshold && !alert.AlertSent {
        // Send the alert and mark it as sent
        alert.LastAlertTime = time.Now()
        alert.AlertSent = true
        sendAlert(emailConfig, site.Custodians, site.SMSCustodians, site.URL, reason, true)
    }
}



func handleSuccess(site Website, emailConfig Email, successThreshold int) {
    state, _ := alertState.LoadOrStore(site.URL, &AlertState{})
    alert := state.(*AlertState)

    alert.ConsecutiveSuccesses++
    alert.ConsecutiveFails = 0 // Reset failures on success

    if alert.ConsecutiveSuccesses >= successThreshold && alert.AlertSent {
        alert.AlertSent = false // Reset the "alert sent" flag
        sendAlert(emailConfig, site.Custodians, site.SMSCustodians, site.URL, "Website is back up", false)
    }
}


// mail handler
func sendAlert(emailConfig Email, custodians []string, smsCustodians []string, url string, reason string, isDown bool) {
    dialer := gomail.NewDialer(emailConfig.SMTPHost, emailConfig.SMTPPort, emailConfig.Username, emailConfig.Password)

    for _, custodian := range custodians {
        message := gomail.NewMessage()
        message.SetHeader("From", emailConfig.Username)

        // Set only the current custodian as the recipient
        message.SetHeader("To", custodian)

        if isDown {
            message.SetHeader("Subject", fmt.Sprintf("Website Down Alert: %s", url))
            message.SetBody("text/plain", fmt.Sprintf("The website %s is down. Reason: %s", url, reason))
        } else {
            message.SetHeader("Subject", fmt.Sprintf("Website Back Up Notification: %s", url))
            message.SetBody("text/plain", fmt.Sprintf("Good news! The website %s is back up and running.", url))
        }

        if err := dialer.DialAndSend(message); err != nil {
            log.Printf("Failed to send alert email to %s: %v", custodian, err)
        } else {
            log.Printf("Alert email sent to %s", custodian)
        }
    }
     // Send SMS Alerts (Only if smsCustodians exists and is non-empty)
     if len(smsCustodians) > 0 {
        for _, smsCustodian := range smsCustodians {
            smsMessage := gomail.NewMessage()
            smsMessage.SetHeader("From", emailConfig.Username)
            smsMessage.SetHeader("To", smsCustodian)

            if isDown{
                smsMessage.SetBody("text/plain", fmt.Sprintf("ALERT: %s is down: %s", url, reason))
            } else {
                smsMessage.SetBody("text/plain", fmt.Sprintf("ALERT: %s is back up!", url))
            }
            

            if err := dialer.DialAndSend(smsMessage); err != nil {
                log.Printf("Failed to send SMS to %s: %v", smsCustodian, err)
            } else {
                log.Printf("Alert SMS sent to %s", smsCustodian)
            }
        }
    }

}
