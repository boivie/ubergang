package mqtt

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	ugtls "boivie/ubergang/server/tls"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

type MqttProxy struct {
	log           *log.Log
	config        *models.Configuration
	db            *db.DB
	tracker       *Tracker
	certManager   ugtls.TlsManager
	brokerAddress string
}

func New(log *log.Log, config *models.Configuration, db *db.DB, tlsManager ugtls.TlsManager, brokerAddress string) *MqttProxy {
	return &MqttProxy{log, config, db, NewTracker(db, log), tlsManager, brokerAddress}
}

type pendingSubscription struct {
	clientSubscribe *packets.SubscribePacket
	brokerSubscribe *packets.SubscribePacket
}

type ConnectionTracker interface {
	GetActiveConnections() map[string]*ClientConnectionState
	DisconnectClient(clientId string)
}

type connection struct {
	parent       *MqttProxy
	connId       int64
	clientConn   net.Conn
	brokerConn   net.Conn
	clientConfig *models.MqttClient
	acl          *ACL

	// Everything below protected by mutex
	mu                   sync.Mutex
	pendingSubscriptions map[uint16] /*MessageId*/ pendingSubscription
}

func (s *MqttProxy) GetActiveConnections() map[string]*ClientConnectionState {
	return s.tracker.GetConnections()
}

func (s *MqttProxy) DisconnectClient(clientId string) {
	s.tracker.Disconnect(clientId)
}

func (s *MqttProxy) listenTLS(port int) {
	listenAddr := fmt.Sprintf(":%d", port)
	s.log.Infof("Starting MQTT TLS proxy server on %s\n", listenAddr)
	listener, err := net.Listen("tcp", ":8883")
	if err != nil {
		s.log.Fatalf("Failed to listen: %v", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: s.certManager.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}
	tlsListener := tls.NewListener(listener, tlsConfig)
	for {
		clientConn, err := tlsListener.Accept()
		if err != nil {
			os.Exit(-1)
		}
		go s.handleClientConnection(ConnectionTypeMQttTls, clientConn)
	}
}

func (s *MqttProxy) listen(port int) {
	listenAddr := fmt.Sprintf(":%d", port)
	s.log.Infof("Starting MQTT proxy server on %s\n", listenAddr)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		s.log.Errorf("Failed to listen to incoming MQTT at %s\n", listenAddr)
		os.Exit(-1)
	}

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			os.Exit(-1)
		}
		go s.handleClientConnection(ConnectionTypeMQtt, clientConn)
	}
}

func (s *MqttProxy) handleClientConnection(connectionType ConnectionType, clientConn net.Conn) {
	defer func() {
		_ = clientConn.Close()
	}()
	s.log.Infof("mqtt: Accepted %s connection from %s", connectionType, clientConn.RemoteAddr())

	connect, err := handleClientConnect(clientConn)
	if err != nil {
		s.log.Warnf("mqtt: Failed to read CONNECT packet: %v from %s", err, clientConn.RemoteAddr())
		connectionErrorMetric.WithLabelValues("read_connect").Inc()
		return
	}

	password := string(connect.Password)
	acl, clientConfig, err := s.authorizeConnection(connect.Username, password)
	if err != nil {
		s.log.Warnf("mqtt: Failed to authorize connection (%s/%s): %v from %s", connect.Username, password, err, clientConn.RemoteAddr())
		connectionErrorMetric.WithLabelValues("failed_auth").Inc()
		_ = connectionRefused().Write(clientConn)
		return
	}

	// Also connect to the broker before we continue.
	brokerConn, err := net.Dial("tcp", s.brokerAddress)
	if err != nil {
		s.log.Warnf("mqtt: Failed to connect to broker: %v", err)
		connectionErrorMetric.WithLabelValues("connect_broker").Inc()
		_ = serverUnavailable().Write(clientConn)
		return
	}
	defer func() {
		_ = brokerConn.Close()
	}()

	connId := s.tracker.AddConnection(clientConfig.Id, connectionType, clientConn, clientConn.RemoteAddr().String())
	defer s.tracker.RemoveConnection(clientConfig.Id, connId)

	// Authenticated and accepted.
	conn := &connection{
		parent:               s,
		connId:               connId,
		clientConn:           clientConn,
		brokerConn:           brokerConn,
		clientConfig:         clientConfig,
		acl:                  acl,
		pendingSubscriptions: make(map[uint16]pendingSubscription),
	}
	conn.serveClient(connect)
}

func (s *MqttProxy) Start(port int, tlsPort int) {
	go s.tracker.Run()
	go s.listen(port)
	go s.listenTLS(port)
}

func (c *MqttProxy) authorizeConnection(username, password string) (acl *ACL, clientConfig *models.MqttClient, err error) {
	clientConfig, err = c.db.GetMqttClient(username)
	if err != nil {
		return
	}

	if clientConfig.Password != password {
		err = errors.New("invalid password")
		return
	}

	clientProfile, err := c.db.GetMqttProfile(clientConfig.ProfileId)
	if err != nil {
		return
	}

	acl, err = NewACL(clientConfig, clientProfile)
	return
}
