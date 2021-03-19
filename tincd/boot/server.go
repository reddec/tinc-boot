package boot

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/reddec/tinc-boot/tincd/daemon"
	"github.com/reddec/tinc-boot/types"
)

func NewServer(config *daemon.Config, token Token) *Server {
	return &Server{
		config: config,
		token:  token,
	}
}

type Server struct {
	Joined func(info Envelope) // hook to handle arrived join request, executed after response

	config *daemon.Config
	token  Token
}

func (srv *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	const maxPayload = 8192
	// payload - encrypted Envelope, described self node
	// output - JSON map of known hosts

	defer request.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(request.Body, maxPayload))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	var env Envelope

	err = env.Open(srv.token, payload)

	if err != nil {
		http.Error(writer, err.Error(), http.StatusUnauthorized)
		return
	}

	if types.CleanString(env.Name) != env.Name {
		http.Error(writer, "invalid node name", http.StatusUnprocessableEntity)
		return
	}

	err = srv.config.AddHost(env.Name, env.Config)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	hosts, err := srv.config.Hosts()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	plainResponse, err := json.Marshal(hosts)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	encryptedResponse := srv.token.Encrypt(plainResponse)
	writer.Header().Set("Content-Length", strconv.Itoa(len(encryptedResponse)))
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(encryptedResponse)
	if callback := srv.Joined; callback != nil {
		callback(env)
	}
}

type Envelope struct {
	Name   string
	Config []byte
}

func (env *Envelope) Seal(t Token) ([]byte, error) {
	data, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	return t.Encrypt(data), nil
}

func (env *Envelope) Open(t Token, data []byte) error {
	plain, err := t.Decrypt(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, env)
}
