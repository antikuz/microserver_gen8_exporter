package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	urnThermal = "/redfish/v1/Chassis/1/Thermal/"
	urnSession = "/redfish/v1/SessionService/Sessions"
)

type MicroserverGen8Collector struct {
	MicroserverGen8
	authHeader       string
	deleteSessionURI string
	client           *http.Client
}

type MicroserverGen8 struct {
	url      string
	login    string
	passwd   string
	insecure bool
}

type Fan struct {
	CurrentReading int    `json:"CurrentReading"`
	Name           string `json:"FanName"`
	Status         `json:"status,omitempty"`
}

type FansArray struct {
	Fans []Fan `json:"Fans"`
}

type TemperatureSensor struct {
	CurrentReading         float64    `json:"CurrentReading"`
	Name                   string `json:"Name"`
	Status                 `json:"status,omitempty"`
	UpperThresholdCritical float64 `json:"UpperThresholdCritical"`
	UpperThresholdFatal    float64 `json:"UpperThresholdFatal"`
}

type Status struct {
	Health string `json:"health,omitempty"`
	State  string `json:"state"`
}

type TemperatureSensorsArray struct {
	Sensors []TemperatureSensor `json:"temperatures"`
}

var (
	sensorStatusDesc = prometheus.NewDesc(
		"microserver_gen8_sensor_state",
		"Temperatures",
		[]string{"name", "health"}, nil,
	)
	sensorTemperatureDesc = prometheus.NewDesc(
		"microserver_gen8_sensor_temperature",
		"Temperatures",
		[]string{"name"}, nil,
	)
	sensorTemperatureUpperCriticalDesc = prometheus.NewDesc(
		"microserver_gen8_sensor_temperature_critical",
		"Temperatures",
		[]string{"name"}, nil,
	)
	sensorTemperatureUpperFatalDesc = prometheus.NewDesc(
		"microserver_gen8_sensor_temperature_fatal",
		"Temperatures",
		[]string{"name"}, nil,
	)
	fanUsageDesc = prometheus.NewDesc(
		"microserver_fan_usage",
		"fan usage",
		[]string{"name", "health", "state"}, nil,
	)
)

func (m *MicroserverGen8Collector) getSession() (string, string) {
	postBody, err := json.Marshal(map[string]string{
		"UserName": m.login,
		"Password": m.passwd,
	})
	if err != nil {
		log.Fatalf("Error marshal microserver credentials due to error: %v", err)
	}

	payload := bytes.NewBuffer(postBody)
	url := m.url + urnSession
	request, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Add("content-type", "application/json")
	res, err := m.client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
		}
		log.Fatalf("Error status: %d, body: %s\n", res.StatusCode, body)
	}
	authHeader := res.Header.Get("X-Auth-Token")
	deleteSessionURI := res.Header.Get("Location")

	return authHeader, deleteSessionURI
}

func (m *MicroserverGen8Collector) close() error {
	fmt.Println("start close")
	request, err := http.NewRequest("DELETE", m.deleteSessionURI, nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("content-type", "application/json")
	request.Header.Add("X-Auth-Token", m.authHeader)
	res, err := m.client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("Error Request: %s\n, status: %d\n", m.deleteSessionURI, res.StatusCode)
	}

	return nil
}

func (m *MicroserverGen8Collector) GetRESTApiData() (FansArray, TemperatureSensorsArray) {
	url := m.url + urnThermal
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Add("content-type", "application/json")
	request.Header.Add("X-Auth-Token", m.authHeader)
	res, err := m.client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("Error Request: %v\n, status: %d\n", request, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	fansArray := FansArray{}
	err = json.Unmarshal(body, &fansArray)
	if err != nil {
		log.Fatalln(err)
	}

	temperatureSensorsArray := TemperatureSensorsArray{}
	err = json.Unmarshal(body, &temperatureSensorsArray)
	if err != nil {
		log.Fatalln(err)
	}

	return fansArray, temperatureSensorsArray
}

func NewMicroserverGen8Collector(ms MicroserverGen8) *MicroserverGen8Collector {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ms.insecure,
			},
		},
	}
	msc := &MicroserverGen8Collector{
		MicroserverGen8:  ms,
		authHeader:       "",
		deleteSessionURI: "",
		client:           client,
	}
	authHeader, deleteSessionURI := msc.getSession()
	fmt.Printf("authHeader: %s, deletesessionURI: %s\n", authHeader, deleteSessionURI)
	msc.authHeader = authHeader
	msc.deleteSessionURI = deleteSessionURI

	return msc
}

func (m MicroserverGen8Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(m, ch)
}

func (m MicroserverGen8Collector) Collect(ch chan<- prometheus.Metric) {
	fansArray, temperatureSensorsArray := m.GetRESTApiData()
	for _, fan := range fansArray.Fans {
		ch <- prometheus.MustNewConstMetric(
			fanUsageDesc,
			prometheus.GaugeValue,
			float64(fan.CurrentReading),
			fan.Name, fan.Health, fan.State,
		)
	}
	for _, sensor := range temperatureSensorsArray.Sensors {
		var state float64
		if  sensor.State == "Enabled" {
			state = 1
		} else {
			state = 0
		}

		ch <- prometheus.MustNewConstMetric(
			sensorStatusDesc,
			prometheus.GaugeValue,
			state,
			sensor.Name,
			sensor.Health,
		)
		ch <- prometheus.MustNewConstMetric(
			sensorTemperatureDesc,
			prometheus.GaugeValue,
			sensor.CurrentReading,
			sensor.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			sensorTemperatureUpperCriticalDesc,
			prometheus.GaugeValue,
			sensor.UpperThresholdCritical,
			sensor.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			sensorTemperatureUpperFatalDesc,
			prometheus.GaugeValue,
			sensor.UpperThresholdCritical,
			sensor.Name,
		)
	}
}

func main() {
	cfg := GetConfig()
	server := MicroserverGen8{
		url:      cfg.Url,
		login:    cfg.Login,
		passwd:   cfg.Passwd,
		insecure: cfg.Insecure,
	}

	msCollector := NewMicroserverGen8Collector(server)
	defer msCollector.close()

	reg := prometheus.NewPedanticRegistry()

	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
		msCollector,
	)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
