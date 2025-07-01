package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// pulled from https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#MustRegister
// we'll add more here later if i feel like they're necessary
type metrics struct {
	power prometheus.Gauge
}

// environment variables
var (
	shellyPassword = os.Getenv("SHELLY_PASSWORD")
	shellyHost     = "192.168.1.36" //os.Getenv("SHELLY_HOST")
	port           = os.Getenv("LISTEN_PORT")
	shellyURL      = fmt.Sprintf("http://%s/rpc/Shelly.GetStatus", shellyHost)
)

// set port to 2112 if port not set
func init() {
	if port == "" {
		port = "2112"
	}
}

// JSON structure returned from GET on the shelly plug
// i'm going to put some possible extra ones in here that i may use, but for now i just actually want apower
type ShellyStatus struct {
	Switch0 struct {
		Apower  float64 `json:"apower"`  // active power
		Voltage float64 `json:"voltage"` // line voltage
		Current float64 `json:"current"` // current in amperes
	} `json:"switch:0"`
	Sys struct {
		Uptime int64 `json:"uptime"` // uptime in... something?  seconds?
	} `json:"sys"`
	Temperature struct {
		TC float64 `json:"tC"` // temperature celcius
	} `json:"temperature"`
}

// make new metrics (also pulled from docs)
// i will add more metrics here if i feel like it later
func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		power: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_power_watts",
			Help: "Current power consumption in watts.",
		}),
	}
	reg.MustRegister(m.power)
	return m
}

// fetches metrics from shellyURL with optional auth, parses them, and updates the power metric
func scrapeShelly(m metrics) error {
	// build request
	req, err := http.NewRequest("GET", shellyURL, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// add basic auth if password is set
	if shellyPassword != "" {
		req.SetBasicAuth("user", shellyPassword) // username is arbitrary and ignored
	}

	// fetch data with request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching data from Shelly: %w", err)
	}
	defer resp.Body.Close()

	// read response into body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// unmarshal body to json
	var meter ShellyStatus
	if err := json.Unmarshal(body, &meter); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	// set power usage
	m.power.Set(meter.Switch0.Apower)

	return nil
}

func main() {
	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// Create new metrics and register them using the custom registry.
	m := NewMetrics(reg)

	// continuously scrape shelly metrics every 15 seconds and update value in separate goroutine
	go func() {
		for {
			if err := scrapeShelly(*m); err != nil {
				log.Printf("scrapeShelly error: %v", err)
			}
			time.Sleep(15 * time.Second)
		}
	}()

	// usually I use gin but this is directly from the prometheus guide
	// and I don't see anything wrong with it
	// Expose metrics and custom registry via an HTTP server
	// using the HandleFor function. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Printf("Serving metrics at :%v/metrics", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
