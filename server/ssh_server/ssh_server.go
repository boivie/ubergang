package ssh_server

import (
	"boivie/ubergang/server/backends"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"

	"github.com/dop251/goja"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// https://datatracker.ietf.org/doc/html/rfc4254#section-7.2
type localForwardChannelData struct {
	DestAddr string
	DestPort uint32

	OriginAddr string
	OriginPort uint32
}

// https://datatracker.ietf.org/doc/html/rfc4254#section-7.1
type remoteForwardRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardSuccess struct {
	BindPort uint32
}

type remoteForwardChannelData struct {
	DestAddr   string
	DestPort   uint32
	OriginAddr string
	OriginPort uint32
}

type contextKey struct {
	name string
}

var ContextKey = &contextKey{"ug_ctx"}

type ugCtx struct {
	SshKeyID      string
	SshKeyValid   bool
	addedBackends []*roamingBackend
}

func getCtx(c ssh.Context) *ugCtx {
	if c.Value(ContextKey) == nil {
		c.SetValue(ContextKey, &ugCtx{})
	}
	return c.Value(ContextKey).(*ugCtx)
}

type SSHServer struct {
	log      *log.Log
	config   *models.Configuration
	db       *db.DB
	backends *backends.BackendManager
}

type roamingConn struct {
	ch gossh.Channel
}

func (c *roamingConn) Read(b []byte) (n int, err error)   { return c.ch.Read(b) }
func (c *roamingConn) Write(b []byte) (n int, err error)  { return c.ch.Write(b) }
func (c *roamingConn) Close() error                       { return c.ch.Close() }
func (c *roamingConn) LocalAddr() net.Addr                { return net.Addr(nil) }
func (c *roamingConn) RemoteAddr() net.Addr               { return net.Addr(nil) }
func (c *roamingConn) SetDeadline(t time.Time) error      { return nil }
func (c *roamingConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *roamingConn) SetWriteDeadline(t time.Time) error { return nil }

type roamingBackend struct {
	conn     *gossh.ServerConn
	log      *log.Log
	bindAddr string
	host     string
}

func (b *roamingBackend) Type() string              { return "roaming" }
func (b *roamingBackend) Host() string              { return b.host }
func (b *roamingBackend) Headers() []*models.Header { return []*models.Header{} }
func (b *roamingBackend) NeedsAuth() bool           { return true }
func (b *roamingBackend) URL() *url.URL {
	u, _ := url.Parse("http://" + b.host)
	return u
}
func (b *roamingBackend) JsScript() *goja.Program {
	return nil
}
func (b *roamingBackend) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	payload := gossh.Marshal(&remoteForwardChannelData{
		DestAddr:   b.bindAddr,
		DestPort:   80,
		OriginAddr: "test.example.com",
		OriginPort: uint32(9000),
	})
	ch, reqs, err := b.conn.OpenChannel("forwarded-tcpip", payload)
	if err != nil {
		b.log.Warnf("Failed to open channel: %v", err)
		return nil, err
	}
	go gossh.DiscardRequests(reqs)
	return &roamingConn{ch}, nil
}

func New(log *log.Log, config *models.Configuration, db *db.DB, backends *backends.BackendManager) *SSHServer {
	return &SSHServer{log, config, db, backends}
}

func (s *SSHServer) DirectTCPIPHandler(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
	portMapping := localForwardChannelData{}
	if err := gossh.Unmarshal(newChan.ExtraData(), &portMapping); err != nil {
		_ = newChan.Reject(gossh.ConnectionFailed, "error parsing forward data: "+err.Error())
		return
	}
	s.log.Infof("DirectTCPIPHandler: %s:%d -> %s:%d", portMapping.OriginAddr, portMapping.OriginPort, portMapping.DestAddr, portMapping.DestPort)

	if !ctx.Value(ContextKey).(*ugCtx).SshKeyValid {
		_ = newChan.Reject(gossh.ConnectionFailed, "key is expired - run ugcert to renew")
		return
	}

	// TODO: Look this up in database
	host := "test.example.com"

	var dialer net.Dialer
	dconn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		_ = newChan.Reject(gossh.ConnectionFailed, err.Error())
		return
	}
	s.log.Infof("SSH jumping to %s -> %s", portMapping.DestAddr, host)

	ch, reqs, err := newChan.Accept()
	if err != nil {
		_ = dconn.Close()
		return
	}
	go gossh.DiscardRequests(reqs)

	go func() {
		defer func() { _ = ch.Close() }()
		defer func() { _ = dconn.Close() }()
		_, _ = io.Copy(ch, dconn)
	}()
	go func() {
		defer func() { _ = ch.Close() }()
		defer func() { _ = dconn.Close() }()
		_, _ = io.Copy(dconn, ch)
	}()
}

func (s *SSHServer) sshConnectionFailed(conn net.Conn, err error) {
	s.log.Warnf("Failed connection from %s with error: %v", conn.RemoteAddr(), err)
	s.log.Warnf("Failed authentication attempt from %s", conn.RemoteAddr())
}

func (s *SSHServer) handleTunnel(ss ssh.Session) {
	var bindInfo struct {
		Error    string `json:"error"`
		BindAddr string `json:"bind_addr"`
	}
	ctx := ss.Context().Value(ContextKey).(*ugCtx)
	if !ctx.SshKeyValid {
		bindInfo.Error = "You will need to run \"ugcert\" to revalidate your SSH key."
	} else {
		bindInfo.BindAddr = ":1902"
	}
	_ = json.NewEncoder(ss).Encode(&bindInfo)
}

func (s *SSHServer) RemoteForwardHandler(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	reqPayload := remoteForwardRequest{}
	if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
		return false, []byte{}
	}
	s.log.Debugf("RemoteForwardHandler: %s:%d", reqPayload.BindAddr, reqPayload.BindPort)

	if reqPayload.BindPort != 80 {
		s.log.Warnf("Requested to forward port %d - not 80 as expected - denying", reqPayload.BindPort)
		return false, []byte{}
	}

	host := reqPayload.BindAddr + "-roam." + s.config.SiteFqdn
	backend := &roamingBackend{
		conn:     ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn),
		log:      s.log,
		bindAddr: reqPayload.BindAddr,
		host:     host,
	}
	s.backends.AddEphemeral(backend)
	ugCtx := ctx.Value(ContextKey).(*ugCtx)
	ugCtx.addedBackends = append(ugCtx.addedBackends, backend)

	return true, gossh.Marshal(&remoteForwardSuccess{reqPayload.BindPort})
}

func (s *SSHServer) ServeSSH(keyPem []byte, port int) {
	addr := fmt.Sprintf(":%d", port)

	srv := ssh.Server{
		Addr: addr,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			config := &gossh.ServerConfig{}
			config.ServerVersion = "SSH-2.0-Ubergang1"
			return config
		},
		ConnectionFailedCallback: s.sshConnectionFailed,
		PtyCallback:              func(ctx ssh.Context, pty ssh.Pty) bool { return false },
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			return true
		}),
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			s.log.Printf("Asked to reverse proxy %s:%d\n", bindHost, bindPort)
			return true
		},
		Handler: func(ss ssh.Session) {
			s.log.Debugf("Connected %s, command=%v", ss.User(), ss.Command())

			if len(ss.Command()) > 0 && ss.Command()[0] == "tunnel" {
				s.handleTunnel(ss)
				return
			}

			_, _ = io.WriteString(ss, "       __                                       \n")
			_, _ = io.WriteString(ss, ".--.--|  |--.-----.----.-----.---.-.-----.-----.\n")
			_, _ = io.WriteString(ss, "|  |  |  _  |  -__|   _|  _  |  _  |     |  _  |\n")
			_, _ = io.WriteString(ss, "|_____|_____|_____|__| |___  |___._|__|__|___  |\n")
			_, _ = io.WriteString(ss, "                       |_____|           |_____|\n")
			_, _ = io.WriteString(ss, "                                                \n")

			ctx := ss.Context().Value(ContextKey).(*ugCtx)
			backends := ctx.addedBackends

			if !ctx.SshKeyValid {
				_, _ = io.WriteString(ss, "\nYou will need to run \"ugcert\" to revalidate your SSH key.\n\n")
				_ = ss.Close()
				return
			} else if len(backends) == 0 {
				_, _ = io.WriteString(ss, "\nYou have successfully connected, but there were no valid port forwardings or SSH hosts to jump to. Good bye!\n\n")
				_ = ss.Close()
				return
			}

			for _, backend := range ctx.addedBackends {
				_, _ = io.WriteString(ss, "Forwarding https://"+backend.Host()+" -> your computer\n")
			}
			_, _ = io.WriteString(ss, "\n")

			<-ss.Context().Done()
			for _, backend := range ctx.addedBackends {
				s.backends.RemoveEphemeral(backend)
			}
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": s.DirectTCPIPHandler,
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward": s.RemoteForwardHandler,
		},
		PublicKeyHandler: func(ctx ssh.Context, pubKey ssh.PublicKey) bool {
			c := getCtx(ctx)
			sha256Fingerprint := sha256.Sum256(pubKey.Marshal())
			key, err := s.db.GetSshKeyByFingerprint(sha256Fingerprint[:])
			if err != nil {
				s.log.Warnf("Failed to find ssh key")
				return false
			}
			user, err := s.db.GetUserById(key.UserId)
			if err != nil {
				s.log.Warnf("Failed to find user: %s", key.UserId)
				return false
			}
			c.SshKeyID = key.Id
			if key.ExpiresAt == nil {
				c.SshKeyValid = false
			} else {
				c.SshKeyValid = key.ExpiresAt.AsTime().After(time.Now())
			}
			if c.SshKeyValid {
				s.log.Infof("Accepting valid key %s for user %s", key.Name, user.Email)
			} else {
				s.log.Infof("Accepting expired key %s for user %s", key.Name, user.Email)
			}
			return true
		},
	}

	err := srv.SetOption(ssh.HostKeyPEM(keyPem))
	if err != nil {
		s.log.Fatalf("Error parsing SSH server host key: %v", err)
	}

	s.log.Infof("SSH server started on %s", addr)
	s.log.Fatal(srv.ListenAndServe())
}
