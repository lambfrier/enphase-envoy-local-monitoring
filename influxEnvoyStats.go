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
	"github.com/influxdata/influxdb/client/v2"
    "time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

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
	ReadingTime      int64
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

func main() {
	jsonData, err := ioutil.ReadFile(os.Args[1])
	check(err)

    influxaddr, database, username, password := os.Args[2], os.Args[3], os.Args[4], os.Args[5]

	var apiJsonObj struct {
		Production  json.RawMessage
		Consumption json.RawMessage
		Storage     json.RawMessage
	}
	err = json.Unmarshal(jsonData, &apiJsonObj)
    check(err)

	inverters := Inverters{}
	prod_readings := Eim{}
	productionObj := []interface{}{&inverters, &prod_readings}
	err = json.Unmarshal(apiJsonObj.Production, &productionObj)
    check(err)

	fmt.Printf("%d production: %.3f\n", prod_readings.ReadingTime, prod_readings.WNow)

	consumption_readings := []Eim{}
	err = json.Unmarshal(apiJsonObj.Consumption, &consumption_readings)
    check(err)
	for _, eim := range consumption_readings {
		fmt.Printf("%d %s: %.3f\n", eim.ReadingTime, eim.MeasurementType, eim.WNow)
	}

    // Connect to influxdb specified in commandline arguments
    c, err := client.NewHTTPClient(client.HTTPConfig{
        Addr:     influxaddr,
        Username: username,
        Password: password,
    })
    check(err)
    defer c.Close()

    bp, err := client.NewBatchPoints(client.BatchPointsConfig{
        Database:  database,
        Precision: "s",
    })
    check(err)

    readings := append(consumption_readings, prod_readings)
    for _, reading := range readings {
        tags := map[string]string{
            "type": reading.MeasurementType,
        }
        fields := map[string]interface{}{
            "WNow": reading.WNow,
        }
        createdTime := time.Unix(reading.ReadingTime, 0)
        check(err)
        pt, err := client.NewPoint(
            "reading",
            tags,
            fields,
            createdTime,
        )
        check(err)
        bp.AddPoint(pt)
    }

    // Write the batch
    err = c.Write(bp)
    check(err)

    err = c.Close()
    check(err)
}

