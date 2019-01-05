// Get Enphase Envoy Solar production data into InfluxDB

// For options:
// > influxEnvoyStats -h

// API path used by the webpage provided by Envoy is e.g.:
//  http://envoy/production.json?details=1

// David Lamb
// 2018-12

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"io/ioutil"
	"net/http"
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
	envoyHostPtr := flag.String("e", "envoy", "IP or hostname of Envoy")
	influxAddrPtr := flag.String("dba", "http://localhost:8086", "InfluxDB connection address")
	dbNamePtr := flag.String("dbn", "solar", "Influx database name to put readings in")
	dbUserPtr := flag.String("dbu", "user", "DB username")
	dbPwPtr := flag.String("dbp", "pw", "DB password")
	measurementNamePtr := flag.String("m", "readings", "Influx measurement name customisation (table name equivalent)")
	flag.Parse()

	envoyUrl := "http://" + *envoyHostPtr + "/production.json?details=1"
	envoyClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}
	req, err := http.NewRequest(http.MethodGet, envoyUrl, nil)
	check(err)
	resp, err := envoyClient.Do(req)
	check(err)
	jsonData, err := ioutil.ReadAll(resp.Body)
	check(err)

	var apiJsonObj struct {
		Production  json.RawMessage
		Consumption json.RawMessage
		Storage     json.RawMessage
	}
	err = json.Unmarshal(jsonData, &apiJsonObj)
	check(err)

	inverters := Inverters{}
	prodReadings := Eim{}
	productionObj := []interface{}{&inverters, &prodReadings}
	err = json.Unmarshal(apiJsonObj.Production, &productionObj)
	check(err)

	fmt.Printf("%d production: %.3f\n", prodReadings.ReadingTime, prodReadings.WNow)

	consumptionReadings := []Eim{}
	err = json.Unmarshal(apiJsonObj.Consumption, &consumptionReadings)
	check(err)
	for _, eim := range consumptionReadings {
		fmt.Printf("%d %s: %.3f\n", eim.ReadingTime, eim.MeasurementType, eim.WNow)
	}

	// Connect to influxdb specified in commandline arguments
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *influxAddrPtr,
		Username: *dbUserPtr,
		Password: *dbPwPtr,
	})
	check(err)
	defer c.Close()

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  *dbNamePtr,
		Precision: "s",
	})
	check(err)

	readings := append(consumptionReadings, prodReadings)
	for _, reading := range readings {
		tags := map[string]string{
			"type": reading.MeasurementType,
		}
		fields := map[string]interface{}{
			"watts": reading.WNow,
		}
		createdTime := time.Unix(reading.ReadingTime, 0)
		check(err)
		pt, err := client.NewPoint(
			*measurementNamePtr,
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
