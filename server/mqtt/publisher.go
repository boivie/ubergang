package mqtt

import (
	"boivie/ubergang/server/log"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	publishSuccessMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ubermqtt_publisher_success_total",
		Help: "Total number of successful publishes",
	})

	publishErrorMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubermqtt_publisher_errors_total",
		Help: "Total number of publish errors",
	}, []string{"error_type"})

	reconnectMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ubermqtt_publisher_reconnects_total",
		Help: "Total number of reconnection attempts",
	})

	connectionStateMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ubermqtt_publisher_connected",
		Help: "Whether publisher is connected (1) or not (0)",
	})
)

// MQTTPublisher defines the interface for publishing MQTT messages
type MQTTPublisher interface {
	Publish(topic string, payload []byte, qos byte, retain bool) error
	IsConnected() bool
	Close() error
}

// Publisher maintains a persistent MQTT connection for server-side publishing
type Publisher struct {
	log    *log.Log
	client mqtt.Client
	mu     sync.RWMutex
}

// NewPublisher creates a new MQTT publisher and starts connection management
func NewPublisher(log *log.Log, brokerAddress string) MQTTPublisher {
	clientID := fmt.Sprintf("ubergang-server-%s", uuid.New().String()[:8])

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + brokerAddress)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(1 * time.Second)
	opts.SetMaxReconnectInterval(60 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetWriteTimeout(5 * time.Second)

	// Connection state callbacks
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Infof("mqtt/publisher: Connected to broker at %s", brokerAddress)
		connectionStateMetric.Set(1)
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Warnf("mqtt/publisher: Connection lost: %v", err)
		connectionStateMetric.Set(0)
	})

	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		log.Warnf("mqtt/publisher: Reconnecting to broker...")
		reconnectMetric.Inc()
	})

	p := &Publisher{
		log:    log,
		client: mqtt.NewClient(opts),
	}

	// Initial connection attempt
	log.Infof("mqtt/publisher: Connecting to broker at %s", brokerAddress)
	if token := p.client.Connect(); token.Wait() && token.Error() != nil {
		log.Warnf("mqtt/publisher: Initial connection failed: %v (will retry automatically)", token.Error())
		connectionStateMetric.Set(0)
	}

	return p
}

// Publish sends an MQTT message synchronously
func (p *Publisher) Publish(topic string, payload []byte, qos byte, retain bool) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.client.IsConnected() {
		publishErrorMetric.WithLabelValues("not_connected").Inc()
		return fmt.Errorf("not connected to broker")
	}

	token := p.client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(5 * time.Second) {
		publishErrorMetric.WithLabelValues("timeout").Inc()
		return fmt.Errorf("publish timeout")
	}

	if err := token.Error(); err != nil {
		publishErrorMetric.WithLabelValues("publish_failed").Inc()
		return err
	}

	publishSuccessMetric.Inc()
	return nil
}

// IsConnected returns whether the publisher is currently connected
func (p *Publisher) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.client.IsConnected()
}

// Close gracefully shuts down the publisher
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log.Info("mqtt/publisher: Shutting down...")
	p.client.Disconnect(250) // Wait up to 250ms for clean disconnect
	connectionStateMetric.Set(0)
	return nil
}
