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
	power           prometheus.Gauge
	total_power     prometheus.Gauge
	voltage         prometheus.Gauge
	output          prometheus.Gauge
	uptime          prometheus.Gauge
	temperature     prometheus.Gauge
	signal_strength prometheus.Gauge
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
type ShellyStatus struct {
	Switch0 struct {
		Output  bool    `json:"output"`  // switch state (on/off)
		Apower  float64 `json:"apower"`  // active power in watts
		Voltage float64 `json:"voltage"` // line voltage
		Aenergy struct {
			Total float64 `json:"total"` // total energy in watt-hours
		} `json:"aenergy"`
		Temperature struct {
			TC float64 `json:"tC"` // temperature celsius
		} `json:"temperature"`
	} `json:"switch:0"`
	Sys struct {
		Uptime int64 `json:"uptime"` // uptime in seconds
	} `json:"sys"`
	Wifi struct {
		Rssi int `json:"rssi"` // WiFi signal strength in dBm
	} `json:"wifi"`
}

// make new metrics (also pulled from docs)
// i will add more metrics here if i feel like it later
func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		power: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_power_watts",
			Help: "Current power consumption in watts.",
		}),
		total_power: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_energy_total_watthours",
			Help: "Total power consumption of plug",
		}),
		voltage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_voltage_volts",
			Help: "Line voltage in volts",
		}),
		output: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_plug_output",
			Help: "The state of output on the plug.  0 for off, 1 for on",
		}),
		uptime: prometheus.NewGauge(prometheus.GaugeOpts{ // i think it may reset to 0 on reset so keep it as a gague
			Name: "shelly_plug_uptime_seconds",
			Help: "Uptime of the shelly plug",
		}),
		temperature: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_plug_temp_celsius",
			Help: "Temperature of the plug",
		}),
		signal_strength: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shelly_plug_signal_strength_rssi",
			Help: "WiFi signal strength in dBm",
		}),
	}
	reg.MustRegister(m.power)
	reg.MustRegister(m.total_power)
	reg.MustRegister(m.uptime)
	reg.MustRegister(m.output)
	reg.MustRegister(m.temperature)
	reg.MustRegister(m.signal_strength)
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

	// print entire bdy to log (for testing at the moment.  just want to see what all is there)
	log.Printf("Received payload:\n\n%s", string(body))

	// unmarshal body to json
	var meter ShellyStatus
	if err := json.Unmarshal(body, &meter); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	// set metrics
	m.power.Set(meter.Switch0.Apower)
	m.voltage.Set(meter.Switch0.Voltage)
	m.total_power.Set(meter.Switch0.Aenergy.Total)

	// Convert bool to float64 (1.0 for true/on, 0.0 for false/off)
	if meter.Switch0.Output {
		m.output.Set(1.0)
	} else {
		m.output.Set(0.0)
	}

	m.uptime.Set(float64(meter.Sys.Uptime))
	m.temperature.Set(meter.Switch0.Temperature.TC)
	m.signal_strength.Set(float64(meter.Wifi.Rssi))

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
