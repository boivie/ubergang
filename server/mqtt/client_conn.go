package mqtt

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	connectionErrorMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubermqtt_connection_errors_total",
		Help: "The total number of connection errors",
	}, []string{"error"})
	connectionSuccessMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubermqtt_connection_success_total",
		Help: "The total number of connection successes",
	}, []string{"account", "profile"})
	publishMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubermqtt_publish_total",
		Help: "The total number of publishes",
	}, []string{"account", "profile", "topic_class"})
	subscribeTopicsMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubermqtt_subscribe_topics_total",
		Help: "The total number of subscription topics",
	}, []string{"account", "profile", "topic_class"})
	clientPackets = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ubermqtt_packets_client_total",
		Help: "The total number of MQTT packets from clients",
	})
	subscribeMessagesMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubermqtt_subscribe_messages_total",
		Help: "The total number of received messages from subscriptions",
	}, []string{"account", "profile"})
	brokerPackets = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ubermqtt_packets_broker_total",
		Help: "The total number of MQTT packets from broker",
	})
)

func (c *connection) handleBrokerConn() {
	logg := func(format string, args ...interface{}) {
		c.parent.log.Infof("mqtt/c%d/%s/broker: "+format, append([]interface{}{c.connId, c.clientConfig.Id}, args...)...)
	}

	defer func() {
		_ = c.clientConn.Close()
	}()
	defer func() {
		_ = c.brokerConn.Close()
	}()
	for {
		controlPacket, err := packets.ReadPacket(c.brokerConn)
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				c.parent.log.Warnf("c%d: Failed to read packet: %v", c.connId, err)
			}
			return
		}
		brokerPackets.Inc()

		switch packet := controlPacket.(type) {
		case *packets.SubackPacket:
			c.mu.Lock()
			subscription, ok := c.pendingSubscriptions[packet.MessageID]
			c.mu.Unlock()
			if !ok {
				c.parent.log.Warnf("c%d: ERROR: Received unsolicited SUBACK from broker: %s", c.connId, packet.String())
			} else {
				subAck := createClientSubAck(subscription.clientSubscribe, subscription.brokerSubscribe, packet)
				_ = subAck.Write(c.clientConn)
			}

		case *packets.PublishPacket:
			subscribeMessagesMetric.WithLabelValues(c.clientConfig.Id).Inc()
			_ = packet.Write(c.clientConn)
		case *packets.UnsubackPacket:
			_ = packet.Write(c.clientConn)
		case *packets.PingrespPacket:
			_ = packet.Write(c.clientConn)
		case *packets.PubackPacket:
			_ = packet.Write(c.clientConn)
		case *packets.PubrecPacket:
			_ = packet.Write(c.clientConn)
		case *packets.PubrelPacket:
			_ = packet.Write(c.clientConn)
		case *packets.PubcompPacket:
			_ = packet.Write(c.clientConn)
		default:
			logg("Unexpected packet: %v", packet.String())
		}
	}
}
func (c *connection) serveClient(connect *packets.ConnectPacket) {
	logg := func(format string, args ...interface{}) {
		c.parent.log.Infof("mqtt/c%d/%s: "+format, append([]interface{}{c.connId, c.clientConfig.Id}, args...)...)
	}
	logg("Serving client")

	// Send CONNECT to broker
	connect.Username = ""
	connect.Password = []byte{}
	connect.ClientIdentifier = fmt.Sprintf("ug-%s", c.clientConfig.Id)
	if connect.WillFlag {
		willAcl := c.acl.ValidatePublishTopic(connect.WillTopic)
		logg("Will topic set to %s (%s)", connect.WillTopic, ClassificationToString(willAcl))
		if willAcl != ALLOWED {
			// TODO: The alternative is to rewrite the connect message, but then
			// the "remaining length" will need to be updated.
			logg("Aborting connection as will topic not allowed")
			_ = serverUnavailable().Write(c.clientConn)
			return
		}
	}
	_ = connect.Write(c.brokerConn)

	controlPacket, err := packets.ReadPacket(c.brokerConn)
	if err != nil {
		logg("Failed to send CONNECT to broker: %v", err)
		_ = serverUnavailable().Write(c.clientConn)
		return
	}
	packet, ok := controlPacket.(*packets.ConnackPacket)
	if !ok {
		logg("Failed to read CONNACK from broker: %v", err)
		_ = serverUnavailable().Write(c.clientConn)
		return
	}
	_ = packet.Write(c.clientConn)
	if packet.ReturnCode != 0 {
		logg("CONNACK return code: %d - exiting", packet.ReturnCode)
		return
	}

	connectionSuccessMetric.WithLabelValues(c.clientConfig.Id, c.clientConfig.ProfileId).Inc()
	// Read from broker
	go c.handleBrokerConn()

	for {
		controlPacket, err := packets.ReadPacket(c.clientConn)
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				logg("Failed to read packet: %v", err)
			}
			return
		}
		clientPackets.Inc()

		switch packet := controlPacket.(type) {
		case *packets.SubscribePacket:
			var allowed []string
			for _, topic := range packet.Topics {
				classification := c.acl.ValidateSubscribeTopic(topic)
				if classification == ALLOWED {
					allowed = append(allowed, topic)
				}
				subscribeTopicsMetric.WithLabelValues(c.clientConfig.Id, c.clientConfig.ProfileId, ClassificationToString(classification)).Inc()
				logg("Subscribe to %s (%s)", topic, ClassificationToString(classification))
			}
			brokerSubscribe := createBrokerSubscribe(packet, allowed)
			if len(brokerSubscribe.Topics) == 0 {
				log.Println("Subscribe contains no valid topics - ignoring")
				// No valid topics to subscribe to. Don't proxy through broker.
				ack := createClientSubAck(packet, brokerSubscribe, nil)
				_ = ack.Write(c.clientConn)
			} else {
				// When the SUBACK response comes from the broker, send a modified
				// SUBACK packet to the client.
				c.mu.Lock()
				c.pendingSubscriptions[packet.MessageID] = pendingSubscription{
					clientSubscribe: packet,
					brokerSubscribe: brokerSubscribe,
				}
				c.mu.Unlock()
				_ = brokerSubscribe.Write(c.brokerConn)
			}

		case *packets.UnsubscribePacket:
			// TODO: Only unsubscribe from topics the client has really subscribed to.
			_ = packet.Write(c.brokerConn)

		case *packets.PublishPacket:
			class := c.acl.ValidatePublishTopic(packet.TopicName)
			if class == BLOCKED {
				packet.TopicName = "_ug/pub/blocked/" + c.clientConfig.Id + "/" + packet.TopicName
			}
			publishMetric.WithLabelValues(c.clientConfig.Id, c.clientConfig.ProfileId, ClassificationToString(class)).Inc()
			_ = packet.Write(c.brokerConn)

		case *packets.PingreqPacket:
			_ = packet.Write(c.brokerConn)
		case *packets.PubackPacket:
			_ = packet.Write(c.brokerConn)
		case *packets.PubrecPacket:
			_ = packet.Write(c.brokerConn)
		case *packets.PubrelPacket:
			_ = packet.Write(c.brokerConn)
		case *packets.PubcompPacket:
			_ = packet.Write(c.brokerConn)
		case *packets.DisconnectPacket:
			_ = packet.Write(c.brokerConn)
			logg("Received DISCONNECT - closing connection")
			return

		default:
			logg("Unexpected packet")
		}
	}
}

func createBrokerSubscribe(clientSub *packets.SubscribePacket, topics []string) *packets.SubscribePacket {
	getQos := func(topic string) byte {
		for idx, packet_topic := range clientSub.Topics {
			if packet_topic == topic {
				return clientSub.Qoss[idx]
			}
		}
		return 0
	}

	brokerSub := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
	brokerSub.MessageID = clientSub.MessageID
	for _, topic := range topics {
		brokerSub.Topics = append(brokerSub.Topics, topic)
		brokerSub.Qoss = append(brokerSub.Qoss, getQos(topic))
	}
	return brokerSub
}

func handleClientConnect(conn net.Conn) (*packets.ConnectPacket, error) {
	controlPacket, err := packets.ReadPacket(conn)
	if err != nil {
		return nil, err
	}
	switch packet := controlPacket.(type) {
	case *packets.ConnectPacket:
		return packet, nil
	default:
		return nil, errors.New("first packet wasn't CONNECT - closing")
	}
}

func connectionRefused() *packets.ConnackPacket {
	response := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	response.ReturnCode = 5 // 0x05 Connection Refused, not authorized
	return response
}

func serverUnavailable() *packets.ConnackPacket {
	response := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	response.ReturnCode = 3 // 0x03 Connection Refused, Server unavailable
	return response
}

func createClientSubAck(clientSub *packets.SubscribePacket, brokerSub *packets.SubscribePacket, brokerAck *packets.SubackPacket) *packets.SubackPacket {
	brokerGrantedQoss := make(map[string]byte)
	if brokerAck != nil {
		for idx, topic := range brokerSub.Topics {
			if idx >= len(brokerAck.ReturnCodes) {
				log.Printf("ERROR: Broker SUBACK doesn't contain return codes for all requested topics\n")
				brokerGrantedQoss[topic] = 0x80
			} else {
				brokerGrantedQoss[topic] = brokerAck.ReturnCodes[idx]
			}
		}
	}

	// Create the modified SUBACK
	clientAck := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	clientAck.MessageID = clientSub.MessageID
	for idx, topic := range clientSub.Topics {
		if qos, ok := brokerGrantedQoss[topic]; ok {
			clientAck.ReturnCodes = append(clientAck.ReturnCodes, qos)
		} else {
			// For non-granted and non-requested subscriptions, pretend to the
			// client that it was granted at the requested QOS level
			clientAck.ReturnCodes = append(clientAck.ReturnCodes, clientSub.Qoss[idx])
		}
	}
	return clientAck
}
