package mqtt

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ConnectionType string

const (
	ConnectionTypeMQtt    ConnectionType = "mqtt"
	ConnectionTypeMQttTls ConnectionType = "mqtt-tls"
)

type ConnectedInfo struct {
	RemoteAddr     string
	ConnectedAt    time.Time
	ConnectionType ConnectionType
}

type DisconnectedInfo struct {
	RemoteAddr     string
	DisconnectedAt *time.Time
}

type ClientConnectionState struct {
	Connected    *ConnectedInfo
	Disconnected *DisconnectedInfo
}

type addConnectionMsg struct {
	clientId       string
	connectionType ConnectionType
	conn           net.Conn
	remoteAddr     string
	connIdChan     chan int64
}

type removeConnectionMsg struct {
	clientId string
	connId   int64
}

type getConnectionsMsg struct {
	response chan map[string]*ClientConnectionState
}

type disconnectMsg struct {
	clientId string
}

type Tracker struct {
	db                   *db.DB
	log                  *log.Log
	addConnectionChan    chan addConnectionMsg
	removeConnectionChan chan removeConnectionMsg
	getConnectionsChan   chan getConnectionsMsg
	disconnectChan       chan disconnectMsg
}

var (
	connectedClientsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ubermqtt_client_connected",
		Help: "Connected clients, by account",
	}, []string{"account", "profile"})
)

func NewTracker(db *db.DB, log *log.Log) *Tracker {
	return &Tracker{
		db:                   db,
		log:                  log,
		addConnectionChan:    make(chan addConnectionMsg),
		removeConnectionChan: make(chan removeConnectionMsg),
		getConnectionsChan:   make(chan getConnectionsMsg),
		disconnectChan:       make(chan disconnectMsg),
	}
}

type connectedInfo struct {
	clientId       string
	remoteAddr     string
	conn           net.Conn
	connId         int64
	connectedAt    time.Time
	connectionType ConnectionType
}

type disconnectedInfo struct {
	remoteAddr     string
	disconnectedAt time.Time
}

type clientInfo struct {
	connected    *connectedInfo
	disconnected *disconnectedInfo
}

func (t *Tracker) Run() {
	infos := make(map[string]clientInfo)
	ticker := time.NewTicker(10 * time.Second)
	connId := int64(0)
	defer ticker.Stop()

	for {
		select {
		case msg := <-t.addConnectionChan:
			connId++
			// Disconnect any existing connected client of this clientId.
			if info, ok := infos[msg.clientId]; ok && info.connected != nil {
				_ = info.connected.conn.Close()
			}

			infos[msg.clientId] = clientInfo{
				connected: &connectedInfo{
					clientId:       msg.clientId,
					remoteAddr:     msg.remoteAddr,
					conn:           msg.conn,
					connId:         connId,
					connectedAt:    time.Now(),
					connectionType: msg.connectionType,
				},
				disconnected: nil,
			}
			msg.connIdChan <- connId
		case msg := <-t.removeConnectionChan:
			if info, ok := infos[msg.clientId]; ok {
				if info.connected != nil && info.connected.connId == msg.connId {
					infos[msg.clientId] = clientInfo{
						connected: nil,
						disconnected: &disconnectedInfo{
							remoteAddr:     info.connected.remoteAddr,
							disconnectedAt: time.Now(),
						},
					}
				}
			}
		case msg := <-t.getConnectionsChan:
			clientStates := make(map[string]*ClientConnectionState)
			for clientId, info := range infos {
				if info.connected != nil {
					clientStates[clientId] = &ClientConnectionState{
						Connected: &ConnectedInfo{
							RemoteAddr:     info.connected.remoteAddr,
							ConnectedAt:    info.connected.connectedAt,
							ConnectionType: info.connected.connectionType,
						},
						Disconnected: nil,
					}
				} else if info.disconnected != nil {
					clientStates[clientId] = &ClientConnectionState{
						Connected: nil,
						Disconnected: &DisconnectedInfo{
							RemoteAddr:     info.disconnected.remoteAddr,
							DisconnectedAt: &info.disconnected.disconnectedAt,
						},
					}
				}
			}
			msg.response <- clientStates

		case msg := <-t.disconnectChan:
			if info, ok := infos[msg.clientId]; ok && info.connected != nil {
				_ = info.connected.conn.Close()
				// This will trigger removeConnection above.
			}
		case <-ticker.C:
			clients := t.db.ListMqttClients()
			for _, client := range clients {
				if info, ok := infos[client.Id]; ok && info.connected != nil {
					connectedClientsMetric.WithLabelValues(client.Id, client.ProfileId).Set(1)
				} else {
					connectedClientsMetric.WithLabelValues(client.Id, client.ProfileId).Set(0)
				}
			}
		}
	}
}

func (t *Tracker) AddConnection(clientId string, connectionType ConnectionType, conn net.Conn, remoteAddr string) int64 {
	connIdChan := make(chan int64)
	t.addConnectionChan <- addConnectionMsg{clientId, connectionType, conn, remoteAddr, connIdChan}
	return <-connIdChan
}

func (t *Tracker) RemoveConnection(clientId string, connId int64) {
	t.removeConnectionChan <- removeConnectionMsg{clientId, connId}
}

func (t *Tracker) GetConnections() map[string]*ClientConnectionState {
	responseChan := make(chan map[string]*ClientConnectionState)
	t.getConnectionsChan <- getConnectionsMsg{response: responseChan}
	return <-responseChan
}

func (t *Tracker) Disconnect(clientId string) {
	t.disconnectChan <- disconnectMsg{clientId}
}
