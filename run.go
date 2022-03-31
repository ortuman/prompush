package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
)

type pusher struct {
	host               string
	user               string
	password           string
	exemplarLabelName  string
	exemplarLabelValue string
}

func runPush(host, username, password, exemplarLabelName, exemplarLabelValue string) error {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	r := pusher{
		host:               host,
		user:               username,
		password:           password,
		exemplarLabelName:  exemplarLabelName,
		exemplarLabelValue: exemplarLabelValue,
	}

	if err := r.loop(); err != nil {
		return err
	}
	for {
		select {
		case <-ticker.C:
			if err := r.loop(); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (p *pusher) loop() error {
	now := time.Now()
	series, _ := p.generateSeries("prompush_series", now,
		prompb.Label{Name: "job", Value: "prompush"},
		prompb.Label{Name: "user", Value: p.user},
	)
	if err := p.pushTimeSeries(series); err != nil {
		return err
	}
	log.Printf("pushed series: %v...\n", series)
	return nil
}

func (p *pusher) pushTimeSeries(timeseries []prompb.TimeSeries) error {
	// Create write request
	data, err := proto.Marshal(&prompb.WriteRequest{Timeseries: timeseries})
	if err != nil {
		return err
	}

	// Create HTTP request
	compressed := snappy.Encode(nil, data)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/v1/push", p.host), bytes.NewReader(compressed))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.SetBasicAuth(p.user, p.password)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Execute HTTP request
	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("unexpected response status code: %d", res.StatusCode)
	}
	return nil
}

func (p *pusher) generateSeries(name string, ts time.Time, additionalLabels ...prompb.Label) (series []prompb.TimeSeries, vector model.Vector) {
	tsMillis := timeToMilliseconds(ts)
	value := rand.Float64()

	lbls := append(
		[]prompb.Label{
			{Name: labels.MetricName, Value: name},
		},
		additionalLabels...,
	)

	// Generate the series
	series = append(series, prompb.TimeSeries{
		Labels: lbls,
		Exemplars: []prompb.Exemplar{
			{
				Value:     value,
				Timestamp: tsMillis,
				Labels: []prompb.Label{
					{Name: p.exemplarLabelName, Value: p.exemplarLabelValue},
				},
			},
		},
		Samples: []prompb.Sample{
			{Value: value, Timestamp: tsMillis},
		},
	})

	// Generate the expected vector when querying it
	metric := model.Metric{}
	metric[labels.MetricName] = model.LabelValue(name)
	for _, lbl := range additionalLabels {
		metric[model.LabelName(lbl.Name)] = model.LabelValue(lbl.Value)
	}

	vector = append(vector, &model.Sample{
		Metric:    metric,
		Value:     model.SampleValue(value),
		Timestamp: model.Time(tsMillis),
	})

	return
}

func timeToMilliseconds(t time.Time) int64 {
	// Convert to seconds.
	sec := float64(t.Unix()) + float64(t.Nanosecond())/1e9

	// Parse seconds.
	s, ns := math.Modf(sec)

	// Round nanoseconds part.
	ns = math.Round(ns*1000) / 1000

	// Convert to millis.
	return (int64(s) * 1e3) + (int64(ns * 1e3))
}
