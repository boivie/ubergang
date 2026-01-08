package main

import (
	"boivie/ubergang/server/api"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	mrand "math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/pkg/browser"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RrandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[mrand.Intn(len(letters))]
	}
	return string(b)
}

func getConfigFilename(filename string) (string, error) {
	if filename != "" {
		return filename, nil
	}
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirname, ".ssh", "ubergang.yaml"), nil
}

func readConfig(filename string) (*ConfigFile, error) {
	filename, err := getConfigFilename(filename)
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			contents = []byte{}
		} else {
			return nil, err
		}
	}

	var config ConfigFile
	err = yaml.Unmarshal(contents, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func writeConfig(filename string, cfg *ConfigFile) error {
	filename, err := getConfigFilename(filename)
	if err != nil {
		return err
	}
	contents, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, contents, 0600)
}

// The inverse of parseOpenSSHPrivateKey in https://github.com/golang/crypto/blob/master/ssh/keys.go
func writeOpenSSHPrivateKey(key ed25519.PrivateKey, name string) []byte {
	const magic = "openssh-key-v1\x00"

	var w struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}

	pk1 := struct {
		Check1  uint32
		Check2  uint32
		Keytype string
		// KeyAlgoED25519
		Pub     []byte
		Priv    []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{}

	ci := mrand.Uint32()
	pk1.Check1 = ci
	pk1.Check2 = ci
	pk1.Keytype = ssh.KeyAlgoED25519

	pk, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return nil
	}
	pubKey := []byte(pk)
	pk1.Pub = pubKey
	pk1.Priv = []byte(key)
	pk1.Comment = name

	bs := 8
	blockLen := len(ssh.Marshal(pk1))
	padLen := (bs - (blockLen % bs)) % bs
	pk1.Pad = make([]byte, padLen)

	for i := 0; i < padLen; i++ {
		pk1.Pad[i] = byte(i + 1)
	}

	prefix := []byte{0x0, 0x0, 0x0, 0x0b}
	prefix = append(prefix, []byte(ssh.KeyAlgoED25519)...)
	prefix = append(prefix, []byte{0x0, 0x0, 0x0, 0x20}...)

	w.CipherName = "none"
	w.KdfName = "none"
	w.KdfOpts = ""
	w.NumKeys = 1
	w.PubKey = append(prefix, pubKey...)
	w.PrivKeyBlock = ssh.Marshal(pk1)

	return append([]byte(magic), ssh.Marshal(w)...)
}

type ServerInfo struct {
	Host      string `yaml:"host"`
	KeyID     string `yaml:"key_id"`
	KeySecret string `yaml:"key_secret"`
	KeyName   string `yaml:"key_name"`
}

type ConfigFile struct {
	Servers []ServerInfo `yaml:"servers"`
}

func updateKey(cCtx *cli.Context) error {
	config, err := readConfig(cCtx.String("config"))
	if err != nil {
		return err
	}
	server := config.Servers[0]

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	publicKey, _ := ssh.NewPublicKey(pubKey)

	name := server.Host + "/" + server.KeyName
	pemKey := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: writeOpenSSHPrivateKey(privKey, name),
	}
	privateKey := pem.EncodeToMemory(pemKey)

	b := &bytes.Buffer{}
	b.WriteString(publicKey.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	_, _ = e.Write(publicKey.Marshal())
	_ = e.Close()
	b.WriteByte(' ')
	b.WriteString(name)
	b.WriteByte('\n')
	authorizedKey := b.Bytes()

	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	fname := filepath.Join(dirname, ".ssh", "id_ug_"+server.KeyID[0:8])
	_ = os.WriteFile(fname, privateKey, 0600)
	_ = os.WriteFile(fname+".pub", authorizedKey, 0644)

	var res api.ApiProposeSshKeyResponse
	req := api.ApiProposeSshKeyRequest{
		KeySecret: server.KeySecret,
		PublicKey: string(authorizedKey),
	}
	err = requests.
		URL("/api/authorized_key/" + server.KeyID).
		Host(server.Host).
		BodyJSON(&req).
		ToJSON(&res).
		Fetch(cCtx.Context)
	if err != nil {
		return err
	}
	fmt.Printf("To continue, visit %s in a browser\n", res.ConfirmUrl)

	_ = browser.OpenURL(res.ConfirmUrl)
	return nil
}

func addKey(cCtx *cli.Context) error {
	u, _ := url.Parse(cCtx.Args().First())
	keyID := strings.TrimPrefix(u.Path, "/")

	var key api.ApiSSHKey
	err := requests.
		URL(fmt.Sprintf("/api/authorized_key/%s", keyID)).
		Host(u.Hostname()).
		ToJSON(&key).
		Fetch(cCtx.Context)
	if err != nil {
		return err
	}

	config, err := readConfig(cCtx.String("config"))
	if err != nil {
		return err
	}
	config.Servers = append(config.Servers, ServerInfo{
		Host:      u.Hostname(),
		KeyID:     key.ID,
		KeySecret: RrandStringBytes(32),
		KeyName:   key.Name,
	})
	err = writeConfig(cCtx.String("config"), config)
	if err != nil {
		return err
	}
	fmt.Println("Key successfully added.")
	return nil
}

func main() {
	app := &cli.App{
		Name:  "boom",
		Usage: "make an explosive entrance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
				EnvVars: []string{"UG_CONFIG"},
			},
		},
		Action: updateKey,
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "add a task to the list",
				Action:  addKey,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
