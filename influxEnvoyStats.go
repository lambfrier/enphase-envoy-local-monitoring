// Get Enphase Envoy Solar production data into influxdb

// e.g. invocation:
// > influxEnvoyStats XX YY

// API path used by the webpage provided by Envoy is e.g.:
//  http://envoy/production.json?details=1

// David Lamb
// 2018-12

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	// "github.com/influxdata/influxdb/client/v2"
)

type EnvoyAPIMeasurement struct {
	Production  json.RawMessage
	Consumption json.RawMessage
	Storage     json.RawMessage
}

type Inverters struct {
	ActiveCount int
}

type Eim struct {
	MeasurementType  string
	ReadingTime      int
	WNow             float64
	WhLifetime       float64
	VarhLeadLifetime float64
	VarhLagLifetime  float64
	VahLifetime      float64
	RmsCurrent       float64
	RmsVoltage       float64
	ReactPwr         float64
	ApprntPwr        float64
	PwrFactor        float64
	WhToday          float64
	WhLastSevenDays  float64
	VahToday         float64
	VarhLeadToday    float64
	VarhLagToday     float64
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	jsonData, err := ioutil.ReadFile(os.Args[1])
	check(err)

	var apiJsonObj struct {
		Production  json.RawMessage
		Consumption json.RawMessage
		Storage     json.RawMessage
	}
	if err := json.Unmarshal(jsonData, &apiJsonObj); err != nil {
		panic(err)
	}

	inverters := Inverters{}
	prod_readings := Eim{}
	productionObj := []interface{}{&inverters, &prod_readings}
	err = json.Unmarshal(apiJsonObj.Production, &productionObj)
	fmt.Printf("%d production: %.3f\n", prod_readings.ReadingTime, prod_readings.WNow)

	consumption_readings := []Eim{}
	err = json.Unmarshal(apiJsonObj.Consumption, &consumption_readings)
	for _, eim := range consumption_readings {
		fmt.Printf("%d %s: %.3f\n", eim.ReadingTime, eim.MeasurementType, eim.WNow)
	}
}
