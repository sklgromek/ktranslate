package vmware_sdwan

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	go_metrics "github.com/kentik/go-metrics"

	"github.com/kentik/ktranslate/pkg/eggs/logger"
	"github.com/kentik/ktranslate/pkg/inputs/http/dialout/vmware_sdwan/models"
	"github.com/kentik/ktranslate/pkg/kt"
)

var json = jsoniter.ConfigFastest

const (
	DefaultPollTimeSec = 60
)

type SDWan struct {
	logger.ContextL

	device   *kt.Device
	config   SDWanConfig
	client   *http.Client
	tr       *http.Transport
	jchfChan chan []*kt.JCHF
}

type SDWanConfig struct {
	Token string `json:"token"`
}

func LaunchHttpPoll(ctx context.Context, log logger.Underlying, registry go_metrics.Registry, jchfChan chan []*kt.JCHF, device *kt.Device) (*SDWan, error) {
	// Set up a http client to talk to the sdwan box.
	sd := SDWan{
		ContextL: logger.NewContextLFromUnderlying(logger.SContext{S: "sdwan " + device.Name}, log),
		device:   device,
		jchfChan: jchfChan,
	}

	// Pull out the parts we want from this config.
	var conf SDWanConfig
	err := json.Unmarshal([]byte(device.HttpConfig), &conf)
	if err != nil {
		return nil, err
	}
	sd.config = conf

	// How to talk to this device.
	sd.tr = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: false},
	}
	sd.client = &http.Client{Transport: sd.tr}

	return &sd, nil
}

func (sd *SDWan) Run(ctx context.Context) {
	dur := sd.device.PollTimeSec
	if dur == 0 {
		dur = DefaultPollTimeSec
	}
	pollTime := time.Second * time.Duration(dur)
	deviceTicker := time.NewTicker(pollTime)
	defer deviceTicker.Stop()

	sd.Infof("running, polling every %v", pollTime)
	for {
		select {
		case <-deviceTicker.C:
			go func() {
				ctxn, cancel := context.WithTimeout(ctx, pollTime) // Limit here so we don't get ahead of ourselvs
				defer cancel()
				sd.getInfo(ctxn)
			}()
		case <-ctx.Done():
			sd.Infof("done")

			return
		}
	}
}

func (sd *SDWan) Close() {}

func (sd *SDWan) poll(ctx context.Context, url string, responce interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Token "+sd.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sd.client.Do(req)
	if err != nil {
		sd.client = &http.Client{Transport: sd.tr}
		return err
	} else {
		defer resp.Body.Close()
		bdy, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		} else {
			if resp.StatusCode >= 400 {
				return fmt.Errorf("There was an error when communicating: %v.", resp.StatusCode)
			} else {
				err = json.Unmarshal(bdy, responce)
				if err != nil {
					return err
				} else {
					return nil // Success.
				}
			}
		}
	}
}

func (sd *SDWan) getInfo(ctx context.Context) error {
	// enterprises -- List all of the customers for this appliance.
	var res models.Enterprise
	err := sd.poll(ctx, "/enterprises", res)
	if err != nil {
		return err
	}

	// getEnterpriseEdges -- List all of the edges for each customer.
	// healthStats -- For Each edge found above, pull latest health stats.

	return nil
}
