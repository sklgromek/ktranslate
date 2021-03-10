package nrm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"sync"
	"time"

	"github.com/kentik/ktranslate/pkg/formats/util"
	"github.com/kentik/ktranslate/pkg/kt"
	"github.com/kentik/ktranslate/pkg/rollup"

	"github.com/kentik/ktranslate/pkg/eggs/logger"
)

const (
	NR_COUNT_TYPE   = "count"
	NR_DIST_TYPE    = "distribution"
	NR_GAUGE_TYPE   = "gauge"
	NR_SUMMARY_TYPE = "summary"
	NR_UNIQUE_TYPE  = "uniqueCount"
)

type NRMFormat struct {
	logger.ContextL
	compression  kt.Compression
	doGz         bool
	lastMetadata map[string]*kt.LastMetadata
	invalids     map[string]bool
	mux          sync.RWMutex
}

type NRMetricSet struct {
	Metrics []NRMetric `json:"metrics"`
}

type NRMetric struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Value      interface{}            `json:"value"`
	Timestamp  int64                  `json:"timestamp"`
	Interval   int64                  `json:"interval.ms"`
	Attributes map[string]interface{} `json:"attributes"`
}

func NewFormat(log logger.Underlying, compression kt.Compression) (*NRMFormat, error) {
	jf := &NRMFormat{
		compression:  compression,
		ContextL:     logger.NewContextLFromUnderlying(logger.SContext{S: "nrmFormat"}, log),
		doGz:         false,
		invalids:     map[string]bool{},
		lastMetadata: map[string]*kt.LastMetadata{},
	}

	switch compression {
	case kt.CompressionNone:
		jf.doGz = false
	case kt.CompressionGzip:
		jf.doGz = true
	default:
		return nil, fmt.Errorf("Invalid compression (%s): format nrm only supports none|gzip", compression)
	}

	return jf, nil
}

func (f *NRMFormat) To(msgs []*kt.JCHF, serBuf []byte) ([]byte, error) {
	ms := NRMetricSet{
		Metrics: make([]NRMetric, 0, len(msgs)*4),
	}
	ct := time.Now().UnixNano() / 1e+6 // Convert to milliseconds
	for _, m := range msgs {
		ms.Metrics = append(ms.Metrics, f.toNRMetric(m, ct)...)
	}

	if len(ms.Metrics) == 0 {
		return nil, nil
	}

	target, err := json.Marshal([]NRMetricSet{ms}) // Has to be an array here, no idea why.
	if err != nil {
		return nil, err
	}

	if !f.doGz {
		return target, nil
	}

	buf := bytes.NewBuffer(serBuf)
	buf.Reset()
	zw, err := gzip.NewWriterLevel(buf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}

	_, err = zw.Write(target)
	if err != nil {
		return nil, err
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (f *NRMFormat) From(raw []byte) ([]map[string]interface{}, error) {
	values := make([]map[string]interface{}, 0)
	return values, nil
}

func (f *NRMFormat) Rollup(rolls []rollup.Rollup) ([]byte, error) {
	if !f.doGz {
		return json.Marshal(rolls)
	}

	serBuf := make([]byte, 0)
	buf := bytes.NewBuffer(serBuf)
	buf.Reset()
	zw, err := gzip.NewWriterLevel(buf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(rolls)
	if err != nil {
		return nil, err
	}

	_, err = zw.Write(b)
	if err != nil {
		return nil, err
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (f *NRMFormat) toNRMetric(in *kt.JCHF, ts int64) []NRMetric {

	switch in.EventType {
	case kt.KENTIK_EVENT_TYPE:
		return f.fromKflow(in, ts)
	case kt.KENTIK_EVENT_SNMP_DEV_METRIC:
		return f.fromSnmpDeviceMetric(in, ts)
	case kt.KENTIK_EVENT_SNMP_INT_METRIC:
		return f.fromSnmpInterfaceMetric(in, ts)
	case kt.KENTIK_EVENT_SYNTH:
		return f.fromKSynth(in, ts)
	case kt.KENTIK_EVENT_SNMP_METADATA:
		return f.fromSnmpMetadata(in, ts)
	default:
		f.mux.Lock()
		defer f.mux.Unlock()
		if !f.invalids[in.EventType] {
			f.Warnf("Invalid EventType: %s", in.EventType)
			f.invalids[in.EventType] = true
		}
	}

	return nil
}

func (f *NRMFormat) fromSnmpMetadata(in *kt.JCHF, ts int64) []NRMetric {

	if in.DeviceName == "" { // Only run if this is set.
		return nil
	}

	lm := util.SetMetadata(in)

	f.mux.Lock()
	defer f.mux.Unlock()
	f.lastMetadata[in.DeviceName] = lm

	return nil
}

func (f *NRMFormat) fromKSynth(in *kt.JCHF, ts int64) []NRMetric {

	var metrics map[string]bool
	var names map[string]string
	switch in.CustomInt["Result Type"] {
	case 0: // Error
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Fetch TTLB | Ping Lost": true,
			"Fetch Size | Ping Min RTT": true, "Ping Max RTT": true, "Ping Avg RTT": true, "Ping Std RTT": true, "Ping Jit RTT": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Sent", "Fetch TTLB | Ping Lost": "Lost",
			"Fetch Size | Ping Min RTT": "MinRTT", "Ping Max RTT": "MaxRTT", "Ping Avg RTT": "AvgRTT", "Ping Std RTT": "StdRTT", "Ping Jit RTT": "JitRTT"}
	case 1: // Timeout
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Fetch TTLB | Ping Lost": true,
			"Fetch Size | Ping Min RTT": true, "Ping Max RTT": true, "Ping Avg RTT": true, "Ping Std RTT": true, "Ping Jit RTT": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Sent", "Fetch TTLB | Ping Lost": "Lost",
			"Fetch Size | Ping Min RTT": "MinRTT", "Ping Max RTT": "MaxRTT", "Ping Avg RTT": "AvgRTT", "Ping Std RTT": "StdRTT", "Ping Jit RTT": "JitRTT"}
	case 2: // Ping
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Fetch TTLB | Ping Lost": true,
			"Fetch Size | Ping Min RTT": true, "Ping Max RTT": true, "Ping Avg RTT": true, "Ping Std RTT": true, "Ping Jit RTT": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Sent", "Fetch TTLB | Ping Lost": "Lost",
			"Fetch Size | Ping Min RTT": "MinRTT", "Ping Max RTT": "MaxRTT", "Ping Avg RTT": "AvgRTT", "Ping Std RTT": "StdRTT", "Ping Jit RTT": "JitRTT"}
	case 3: // Fetch
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Fetch TTLB | Ping Lost": true, "Fetch Size | Ping Min RTT": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Status", "Fetch TTLB | Ping Lost": "TTLB", "Fetch Size | Ping Min RTT": "Size"}
	case 4: // Trace
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Time"}
	case 5: // Knock
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Fetch TTLB | Ping Lost": true,
			"Fetch Size | Ping Min RTT": true, "Ping Max RTT": true, "Ping Avg RTT": true, "Ping Std RTT": true, "Ping Jit RTT": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Sent", "Fetch TTLB | Ping Lost": "Lost",
			"Fetch Size | Ping Min RTT": "MinRTT", "Ping Max RTT": "MaxRTT", "Ping Avg RTT": "AvgRTT", "Ping Std RTT": "StdRTT", "Ping Jit RTT": "JitRTT"}
	case 6: // Query
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Fetch TTLB | Ping Lost": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Time", "Fetch TTLB | Ping Lost": "Code"}
	case 7: // Shake
		metrics = map[string]bool{"Fetch Status | Ping Sent | Trace Time": true, "Lat/Long Dest": true}
		names = map[string]string{"Fetch Status | Ping Sent | Trace Time": "Time", "Lat/Long Dest": "Port"}
	}

	attr := map[string]interface{}{}
	f.mux.RLock()
	util.SetAttr(attr, in, metrics, f.lastMetadata[in.DeviceName])
	f.mux.RUnlock()
	ms := make([]NRMetric, len(metrics))
	i := 0

	for m, _ := range metrics {
		ms[i] = NRMetric{
			Name:       "kentik.synth." + names[m],
			Type:       NR_GAUGE_TYPE,
			Value:      int64(in.CustomInt[m]),
			Timestamp:  ts,
			Attributes: attr,
		}
		i++
	}

	return ms
}

func (f *NRMFormat) fromKflow(in *kt.JCHF, ts int64) []NRMetric {
	// Map the basic strings into here.
	attr := map[string]interface{}{}
	metrics := map[string]bool{"in_bytes": true, "out_bytes": true, "in_pkts": true, "out_pkts": true, "latency_ms": true}
	f.mux.RLock()
	util.SetAttr(attr, in, metrics, f.lastMetadata[in.DeviceName])
	f.mux.RUnlock()
	ms := make([]NRMetric, 0)
	for m, _ := range metrics {
		var value int64
		switch m {
		case "in_bytes":
			value = int64(in.InBytes * uint64(in.SampleRate))
		case "out_bytes":
			value = int64(in.OutBytes * uint64(in.SampleRate))
		case "in_pkts":
			value = int64(in.InPkts * uint64(in.SampleRate))
		case "out_pkts":
			value = int64(in.OutPkts * uint64(in.SampleRate))
		case "latency_ms":
			value = int64(in.CustomInt["APPL_LATENCY_MS"])
		}
		if value > 0 {
			ms = append(ms, NRMetric{
				Name:       "kentik.flow." + m,
				Type:       NR_GAUGE_TYPE,
				Value:      value,
				Timestamp:  ts,
				Attributes: attr,
			})
		}
	}
	return ms
}

func (f *NRMFormat) fromSnmpDeviceMetric(in *kt.JCHF, ts int64) []NRMetric {
	var metrics map[string]bool
	if len(in.CustomMetrics) > 0 {
		metrics = in.CustomMetrics
	} else {
		metrics = map[string]bool{"CPU": true, "MemoryTotal": true, "MemoryUsed": true, "MemoryFree": true, "MemoryUtilization": true, "Uptime": true}
	}
	attr := map[string]interface{}{}
	f.mux.RLock()
	util.SetAttr(attr, in, metrics, f.lastMetadata[in.DeviceName])
	f.mux.RUnlock()
	ms := make([]NRMetric, 0, len(metrics))
	for m, _ := range metrics {
		if _, ok := in.CustomBigInt[m]; ok {
			ms = append(ms, NRMetric{
				Name:       "kentik.snmp." + m,
				Type:       NR_GAUGE_TYPE,
				Value:      int64(in.CustomBigInt[m]),
				Timestamp:  ts,
				Attributes: attr,
			})
		}
	}

	return ms
}

func (f *NRMFormat) fromSnmpInterfaceMetric(in *kt.JCHF, ts int64) []NRMetric {
	var metrics map[string]bool
	if len(in.CustomMetrics) > 0 {
		metrics = in.CustomMetrics
	} else {
		metrics = map[string]bool{"ifHCInOctets": true, "ifHCInUcastPkts": true, "ifHCOutOctets": true, "ifHCOutUcastPkts": true, "ifInErrors": true, "ifOutErrors": true,
			"ifInDiscards": true, "ifOutDiscards": true, "ifHCOutMulticastPkts": true, "ifHCOutBroadcastPkts": true, "ifHCInMulticastPkts": true, "ifHCInBroadcastPkts": true}
	}
	attr := map[string]interface{}{}
	f.mux.RLock()
	defer f.mux.RUnlock()
	util.SetAttr(attr, in, metrics, f.lastMetadata[in.DeviceName])
	ms := make([]NRMetric, 0, len(metrics))
	for m, _ := range metrics {
		if _, ok := in.CustomBigInt[m]; ok {
			ms = append(ms, NRMetric{
				Name:       "kentik.snmp." + m,
				Type:       NR_GAUGE_TYPE,
				Value:      int64(in.CustomBigInt[m]),
				Timestamp:  ts,
				Attributes: attr,
			})
		}
	}

	// Grap capacity utilization if possible.
	if f.lastMetadata[in.DeviceName] != nil {
		if ii, ok := f.lastMetadata[in.DeviceName].InterfaceInfo[in.InputPort]; ok {
			if speed, ok := ii["Speed"]; ok {
				if ispeed, ok := speed.(int32); ok {
					uptimeSpeed := in.CustomBigInt["Uptime"] * (int64(ispeed) * 1000000) // Convert into bits here, from megabits.
					if uptimeSpeed > 0 {
						ms = append(ms, NRMetric{
							Name:       "kentik.snmp.IfInUtilization",
							Type:       NR_GAUGE_TYPE,
							Value:      float64(in.CustomBigInt["ifHCInOctets"]*8*100) / float64(uptimeSpeed),
							Timestamp:  ts,
							Attributes: attr,
						})
					}
				}
			}
		}
		if oi, ok := f.lastMetadata[in.DeviceName].InterfaceInfo[in.OutputPort]; ok {
			if speed, ok := oi["Speed"]; ok {
				if ispeed, ok := speed.(int32); ok {
					uptimeSpeed := in.CustomBigInt["Uptime"] * (int64(ispeed) * 1000000) // Convert into bits here, from megabits.
					if uptimeSpeed > 0 {
						ms = append(ms, NRMetric{
							Name:       "kentik.snmp.IfOutUtilization",
							Type:       NR_GAUGE_TYPE,
							Value:      float64(in.CustomBigInt["ifHCOutOctets"]*8*100) / float64(uptimeSpeed),
							Timestamp:  ts,
							Attributes: attr,
						})
					}
				}
			}
		}
	}

	return ms
}
