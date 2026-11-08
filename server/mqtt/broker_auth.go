package mqtt

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// BrokerCredentials is how ubergang authenticates to the upstream broker. The
// zero value means connecting anonymously.
type BrokerCredentials struct {
	Username string
	Password string
}

// LoadBrokerCredentials builds the upstream credentials from the
// --mqtt-username and --mqtt-password-file flags. The password is read from a
// file so it isn't visible in the process list. Both or neither must be set.
func LoadBrokerCredentials(username, passwordFile string) (BrokerCredentials, error) {
	if username == "" && passwordFile == "" {
		return BrokerCredentials{}, nil
	}
	if username == "" || passwordFile == "" {
		return BrokerCredentials{}, errors.New("--mqtt-username and --mqtt-password-file must be given together")
	}
	b, err := os.ReadFile(passwordFile)
	if err != nil {
		return BrokerCredentials{}, fmt.Errorf("reading MQTT password file: %w", err)
	}
	password := strings.TrimRight(string(b), "\r\n")
	if password == "" {
		return BrokerCredentials{}, fmt.Errorf("MQTT password file %s is empty", passwordFile)
	}
	return BrokerCredentials{Username: username, Password: password}, nil
}

// rewriteConnectForBroker turns a client's CONNECT into the one sent upstream:
// the client's own credentials (checked by ubergang) are replaced by
// ubergang's, and the client ID is namespaced. The flags are set together with
// the fields, as packets.ConnectPacket.Write only looks at the flags.
func rewriteConnectForBroker(connect *packets.ConnectPacket, clientID string, creds BrokerCredentials) {
	connect.ClientIdentifier = fmt.Sprintf("ug-%s", clientID)
	connect.Username = creds.Username
	connect.UsernameFlag = creds.Username != ""
	connect.Password = []byte(creds.Password)
	connect.PasswordFlag = connect.UsernameFlag && creds.Password != ""
}
