package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "bobcat"
)

// Exporter structure
type Exporter struct {
	uri   string
	mutex sync.RWMutex
	fetch func(endpoint string) (io.ReadCloser, error)

	bobcatUp											prometheus.Gauge
	bobcatStatus									prometheus.Gauge
	bobcatStatusGap								prometheus.Gauge
	bobcatStatusMinerHeight				prometheus.Gauge
	bobcatStatusBlockchainHeight	prometheus.Gauge
	bobcatStatusEpoch							prometheus.Gauge
	bobcatTemperatureUnit					prometheus.Gauge
	bobcatTemperatureTemp0				prometheus.Gauge
	bobcatTemperatureTemp1				prometheus.Gauge
	totalScrapes									prometheus.Counter
}

// NewExporter function
func NewExporter(uri string, timeout time.Duration) (*Exporter, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var fetch func(endpoint string) (io.ReadCloser, error)
	switch u.Scheme {
	case "http", "https", "file":
		fetch = fetchHTTP(uri, timeout)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}

	return &Exporter{
		uri:   uri,
		fetch: fetch,
		bobcatUp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "up",
				Help:      "The current health of the miner (1 = UP, 0 = DOWN).",
			},
		),
		bobcatStatus: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "status",
				Help:      "The current status of the miner (1 = 'SYNCED', 0 = 'SYNCING').",
			},
		),
		bobcatStatusGap: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "status_gap",
				Help:      "The current blockchain gap in the miner.",
			},
		),
		bobcatStatusMinerHeight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "status_miner_height",
				Help:      "The current blockchain height of the miner.",
			},
		),
		bobcatStatusBlockchainHeight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "status_blockchain_height",
				Help:      "The current blockchain height from the miner.",
			},
		),
		bobcatStatusEpoch: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "status_epoch",
				Help:      "The current epoch of the miner.",
			},
		),
		bobcatTemperatureUnit: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "temperature_unit",
				Help:      "The current unit temperature of the miner. (1 = '°C', 0 = '°F')",
			},
		),		
		bobcatTemperatureTemp0: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "temperature_temp0",
				Help:      "The current temperature (temp0) of the miner.",
			},
		),
		bobcatTemperatureTemp1: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "temperature_temp1",
				Help:      "The current temperature (temp0) of the miner.",
			},
		),
		totalScrapes: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "total_scrapes",
				Help:      "The total number of scrapes.",
			},
		),
	}, nil
}

// Describe function of Exporter
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	ch <- e.bobcatStatus.Desc()
	ch <- e.bobcatStatusGap.Desc()
	ch <- e.bobcatStatusMinerHeight.Desc()
	ch <- e.bobcatStatusBlockchainHeight.Desc()
	ch <- e.bobcatStatusEpoch.Desc()
	ch <- e.bobcatUp.Desc()
	ch <- e.totalScrapes.Desc()
}

// Collect function of Exporter
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	up := e.scrape(ch)

	ch <- prometheus.MustNewConstMetric(e.bobcatUp.Desc(), prometheus.GaugeValue, up)
}

type temperature struct {
	Timestamp string `json:"timestamp"`
	Temp0     int    `json:"temp0"`
	Temp1     int    `json:"temp1"`
	Unit      string `json:"unit"`
}

type status struct {
	Status           string `json:"status"`
	Gap              string `json:"gap"`
	MinerHeight      string `json:"miner_height"`
	BlockchainHeight string `json:"blockchain_height"`
	Epoch            string `json:"epoch"`
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) (up float64) {
	e.totalScrapes.Inc()

	// ENDPOINT: /status.json
	var status status

	bodyStatus, err := e.fetch("/status.json")
	if err != nil {
		log.Errorf("Can't scrape /status.json: %v", err)
		return 0
	}
	defer bodyStatus.Close()

	bodyStatusAll, err := ioutil.ReadAll(bodyStatus)
	if err != nil {
		return 0
	}

	_ = json.Unmarshal([]byte(bodyStatusAll), &status)

	ch <- prometheus.MustNewConstMetric(e.bobcatStatus.Desc(), prometheus.GaugeValue, parseStatus(status.Status))
	ch <- prometheus.MustNewConstMetric(e.bobcatStatusGap.Desc(), prometheus.GaugeValue, parseString(status.Gap))
	ch <- prometheus.MustNewConstMetric(e.bobcatStatusMinerHeight.Desc(), prometheus.GaugeValue, parseString(status.MinerHeight))
	ch <- prometheus.MustNewConstMetric(e.bobcatStatusBlockchainHeight.Desc(), prometheus.GaugeValue, parseString(status.BlockchainHeight))
	ch <- prometheus.MustNewConstMetric(e.bobcatStatusEpoch.Desc(), prometheus.GaugeValue, parseString(status.Epoch))

	// ENDPOINT: /temp.json
	var temperature temperature

	bodyTemp, err := e.fetch("/temp.json")
	if err != nil {
		log.Errorf("Can't scrape /temp.json: %v", err)
		return 0
	}
	defer bodyTemp.Close()

	bodyTempAll, err := ioutil.ReadAll(bodyTemp)
	if err != nil {
		return 0
	}

	_ = json.Unmarshal([]byte(bodyTempAll), &temperature)

	ch <- prometheus.MustNewConstMetric(e.bobcatTemperatureUnit.Desc(), prometheus.GaugeValue, parseTemperature(temperature.Unit))
	ch <- prometheus.MustNewConstMetric(e.bobcatTemperatureTemp0.Desc(), prometheus.GaugeValue, float64(temperature.Temp0))
	ch <- prometheus.MustNewConstMetric(e.bobcatTemperatureTemp1.Desc(), prometheus.GaugeValue, float64(temperature.Temp1))

	return 1
}

func parseString(str string) (float64) {
	value, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
    return 0;
	}
	return float64(value);
}

func parseTemperature(str string) (float64) {
	if(str == "°C") {
		return 1;
	}
	return 0
}

func parseStatus(str string) (float64) {
	if(str == "Synced") {
		return 1;
	}
	return 0
}

func fetchHTTP(uri string, timeout time.Duration) func(endpoint string) (io.ReadCloser, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	return func(endpoint string) (io.ReadCloser, error) {
		resp, err := client.Get(uri + endpoint)
		if err != nil {
			return nil, err
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
		}
		return resp.Body, nil
	}
}

func main() {

	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9857").Envar("BOBCAT_EXPORTER_WEB_LISTEN_ADDRESS").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").Envar("BOBCAT_EXPORTER_WEB_TELEMETRY_PATH").String()
		uri           = kingpin.Flag("bobcat.uri", "URI of Bobcat.").Default("http://localhost:9292").Envar("BOBCAT_EXPORTER_MINER_URI").String()
		timeout       = kingpin.Flag("bobcat.timeout", "Scrape timeout").Default("30s").Envar("BOBCAT_EXPORTER_MINER_TIMEOUT").Duration()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("bobcat_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting bobcat_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter(*uri, *timeout)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("bobcatexporter"))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Bobcat Exporter</title></head><body><h1>Bobcat Exporter</h1><p><a href='` + *metricsPath + `'>Metrics</a></p></body></html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
