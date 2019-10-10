package node

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/reddec/tinc-boot/types"
	"golang.org/x/crypto/chacha20poly1305"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
	"time"
)

const serviceFile = "/etc/systemd/system/tinc-boot"

type Cmd struct {
	Name    string `long:"name" env:"NAME" description:"Self node name"`
	Dir     string `long:"dir" env:"DIR" description:"Configuration directory (including net)" default:"/etc/tinc/dnet"`
	Binding string `long:"binding" env:"BINDING" description:"Public binding address" default:":8655"`
	Token   string `long:"token" env:"TOKEN" description:"Authorization token (used as a encryption key)"`
	Service bool   `long:"service" env:"SERVICE" description:"Generate service file to /etc/systemd/system/tinc-boot-{net}.service"`
	TLSKey  string `long:"tls-key" env:"TLS_KEY" description:"Path to private TLS key"`
	TLSCert string `long:"tls-cert" env:"TLS_CERT" description:"Path to public TLS certificate"`
}

func (cmd *Cmd) Hosts() string              { return filepath.Join(cmd.Dir, "hosts") }
func (cmd *Cmd) HostFile() string           { return filepath.Join(cmd.Hosts(), cmd.Name) }
func (cmd *Cmd) Network() string            { return filepath.Base(cmd.Dir) }
func (cmd *Cmd) Bin() (string, error)       { return exec.LookPath(os.Args[0]) }
func (cmd *Cmd) Directory() (string, error) { return filepath.Abs(cmd.Dir) }
func (cmd *Cmd) TLS() bool                  { return cmd.TLSKey != "" && cmd.TLSCert != "" }

func (cmd *Cmd) Execute(args []string) error {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	if cmd.Token == "" {
		s := sha256.Sum256([]byte(time.Now().String()))
		cmd.Token = hex.EncodeToString(s[:])
		fmt.Println("Generated token:", cmd.Token)
	}
	if cmd.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		cmd.Name = hostname
	}
	cmd.Name = types.CleanString(cmd.Name)

	tokenData := sha256.Sum256([]byte(cmd.Token)) // normalize to 32 bytes
	crypter, err := chacha20poly1305.NewX(tokenData[:])
	if err != nil {
		return err
	}

	validNames := regexp.MustCompile(`[a-z0-9]+`)

	engine.POST("/:hexseed/:name", func(gctx *gin.Context) {
		nounce, err := hex.DecodeString(gctx.Param("hexseed"))
		if err != nil || len(nounce) != chacha20poly1305.NonceSizeX {
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		nodeName := gctx.Param("name")
		if !validNames.MatchString(nodeName) {
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		encrypted, err := gctx.GetRawData()
		if err != nil {
			gctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		data, err := crypter.Open(nil, nounce, encrypted, []byte(nodeName))
		if err != nil {
			log.Println(err)
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		if !validNames.MatchString(nodeName) {
			log.Println(nodeName, err)
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		destFile := filepath.Join(cmd.Hosts(), nodeName)
		if err := WriteFile(destFile, data, 0755); err != nil {
			gctx.Data(http.StatusBadRequest, "text/plain", []byte(err.Error()))
			return
		}

		data, err = ioutil.ReadFile(cmd.HostFile())
		if err != nil {
			log.Println(err)
			gctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		encrypted = crypter.Seal(nil, nounce, data, []byte(cmd.Name))
		gctx.Header("X-Node", cmd.Name)
		gctx.Data(http.StatusOK, "application/octet-stream", encrypted)
	})

	if cmd.Service {
		return cmd.installService()
	}
	if cmd.TLS() {
		log.Println("bootnode on", cmd.Binding, "(TLS)")
		return engine.RunTLS(cmd.Binding, cmd.TLSCert, cmd.TLSKey)
	} else {
		log.Println("bootnode on", cmd.Binding)
		return engine.Run(cmd.Binding)
	}
}

func (cmd *Cmd) installService() error {
	buf := &bytes.Buffer{}
	err := template.Must(template.New("").Parse(`[Unit]
Description=Boot node for Tinc for network {{.Network}}

[Service]
ExecStart={{.Bin}} bootnode --token "{{.Token}}" --binding "{{.Binding}}" --dir "{{.Directory}}" --name "{{.Name}}"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`)).Execute(buf, cmd)
	if err != nil {
		return err
	}
	serviceName := "tinc-boot-" + cmd.Network()
	targetFile := filepath.Join("/etc", "systemd", "system", serviceName+".service")
	err = ioutil.WriteFile(targetFile, buf.Bytes(), 0755)
	if err != nil {
		return err
	}
	fmt.Println("DONE!")
	fmt.Println("invoke command by root:")
	fmt.Println("")
	fmt.Println("     systemctl start " + serviceName)
	fmt.Println("     systemctl enable " + serviceName)
	fmt.Println("")
	return nil
}

func WriteFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
