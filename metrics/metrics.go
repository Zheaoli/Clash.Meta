package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/hashicorp/golang-lru/v2"
	stdprome "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type Metrics struct {
	clashInfo                                  metrics.Gauge
	clashDownloadBytesTotal                    metrics.Gauge
	clashUploadBytesTotal                      metrics.Gauge
	clashActiveConnections                     metrics.Gauge
	clashNetworkTrafficBytesTotal              metrics.Counter
	clashIPAddressAccessTotal                  metrics.Counter
	clashTracingRuleMatchDurationMilliseconds  metrics.Histogram
	clashTracingDNSRequestDurationMilliseconds metrics.Histogram
	clashTracingProxyDialDurationMilliseconds  metrics.Histogram
	cache                                      *lru.Cache[string, string]
}

var DefaultMetrics *Metrics

func init() {
	var err error
	DefaultMetrics, err = NewMetrics()
	if err != nil {
		log.Fatalf("init metrics error: %v", err)
	}
}

func NewMetrics() (*Metrics, error) {
	cache, err := lru.New[string, string](30000)
	if err != nil {
		return nil, err
	}
	return &Metrics{
		clashInfo: prometheus.NewGaugeFrom(stdprome.GaugeOpts{
			Namespace: "clash",
			Name:      "info",
			Help:      "basic info of clash",
		}, []string{"info"}),
		clashDownloadBytesTotal: prometheus.NewGaugeFrom(stdprome.GaugeOpts{
			Namespace: "clash",
			Name:      "download_bytes_total",
			Help:      "total download bytes of clash",
		}, []string{}),
		clashUploadBytesTotal: prometheus.NewGaugeFrom(stdprome.GaugeOpts{
			Namespace: "clash",
			Name:      "upload_bytes_total",
			Help:      "total upload bytes of clash",
		}, []string{}),
		clashActiveConnections: prometheus.NewGaugeFrom(stdprome.GaugeOpts{
			Namespace: "clash",
			Name:      "active_connections",
			Help:      "active connections of clash",
		}, []string{}),
		clashNetworkTrafficBytesTotal: prometheus.NewCounterFrom(stdprome.CounterOpts{
			Namespace: "clash",
			Name:      "network_traffic_bytes_total",
			Help:      "total network traffic bytes of clash policy",
		}, []string{"source", "destination", "policy", "type", "location"}),
		clashIPAddressAccessTotal: prometheus.NewCounterFrom(stdprome.CounterOpts{
			Namespace: "clash",
			Name:      "ip_address_access_total",
			Help:      "total ip address access of clash policy",
		}, []string{"address", "location"}),
		clashTracingRuleMatchDurationMilliseconds: prometheus.NewHistogramFrom(stdprome.HistogramOpts{
			Namespace: "clash",
			Subsystem: "tracing",
			Name:      "rule_match_duration_milliseconds",
			Help:      "rule match duration milliseconds of clash",
			Buckets:   timeBucket(),
		}, []string{}),
		clashTracingDNSRequestDurationMilliseconds: prometheus.NewHistogramFrom(stdprome.HistogramOpts{
			Namespace: "clash",
			Subsystem: "tracing",
			Name:      "dns_request_duration_milliseconds",
			Help:      "dns request duration milliseconds of clash",
			Buckets:   timeBucket(),
		}, []string{"type"}),
		clashTracingProxyDialDurationMilliseconds: prometheus.NewHistogramFrom(stdprome.HistogramOpts{
			Namespace: "clash",
			Subsystem: "tracing",
			Name:      "proxy_dial_duration_milliseconds",
			Help:      "proxy dial duration milliseconds of clash",
			Buckets:   timeBucket(),
		}, []string{"policy"}),
		cache: cache,
	}, nil
}

func (m *Metrics) ClashInfo(version string) {
	m.clashInfo.With("version", version).Add(1)
}

func (m *Metrics) ClashDownloadBytesTotal(bytes float64) {
	m.clashDownloadBytesTotal.Add(bytes)
}

func (m *Metrics) ClashUploadBytesTotal(bytes float64) {
	m.clashUploadBytesTotal.Add(bytes)
}

func (m *Metrics) ClashActiveConnections(connections float64) {
	m.clashActiveConnections.Set(connections)
}

func (m *Metrics) ClashNetworkTrafficBytesTotal(source, destination, policy, t string, bytes float64) {
	go func() {
		location, ok := m.cache.Get(destination)
		if !ok {
			tempLocation, err := GetCode(destination)
			if err != nil {
				log.Printf("ip2region find error: %v", err)
				return
			}
			location = tempLocation
			m.cache.Add(destination, tempLocation)
		}
		m.clashNetworkTrafficBytesTotal.With("source", source, "destination", destination, "policy", policy, "type", t, "location", location).Add(bytes)
	}()
}

func (m *Metrics) ClashIPAddressAccessTotal(address string) {
	go func() {
		location, ok := m.cache.Get(address)
		if !ok {
			tempLocation, err := GetCode(address)
			if err != nil {
				log.Printf("ip2region find error: %v", err)
				return
			}
			location = tempLocation
			m.cache.Add(address, tempLocation)
		}
		m.clashIPAddressAccessTotal.With("address", address, "location", location).Add(1)
	}()
}

func (m *Metrics) ClashTracingRuleMatchDurationMilliseconds(duration float64) {
	m.clashTracingRuleMatchDurationMilliseconds.Observe(duration)
}

func (m *Metrics) ClashTracingDNSRequestDurationMilliseconds(duration float64, t string) {
	m.clashTracingDNSRequestDurationMilliseconds.With("type", t).Observe(duration)
}

func (m *Metrics) ClashTracingProxyDialDurationMilliseconds(duration float64, policy string) {
	m.clashTracingProxyDialDurationMilliseconds.With("policy", policy).Observe(duration)
}

func timeBucket() []float64 {
	var buckets []float64
	for i := 100; i <= 4500; i += 100 {
		buckets = append(buckets, float64(i))
	}
	for i := 4600; i <= 15000; i += 500 {
		buckets = append(buckets, float64(i))
	}
	for i := 16000; i <= 30000; i += 1000 {
		buckets = append(buckets, float64(i))
	}
	return buckets
}
