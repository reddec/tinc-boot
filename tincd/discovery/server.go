package discovery

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/reddec/tinc-boot/tincd/daemon"
	"github.com/reddec/tinc-boot/types"
)

func NewServer(ssd *SSD, config *daemon.Config) http.Handler {
	srv := &server{
		config: config,
		ssd:    ssd,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.getHeaders)                                            // -> [{"Name": "", "Version": 1234}, ...]
	mux.Handle("/host/", http.StripPrefix("/host/", http.HandlerFunc(srv.getOne))) // /host/abc?after=1234

	return mux
}

type server struct {
	config *daemon.Config
	ssd    *SSD
}

func (srv *server) getHeaders(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = srv.ssd.Marshal(writer)
}

func (srv *server) getOne(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	name := types.CleanString(strings.Trim(request.URL.Path, "/ "))
	lastVersion := request.URL.Query().Get("after")
	var version int64
	if v, err := strconv.ParseInt(lastVersion, 10, 64); err == nil {
		version = v
	}

	info, ok := srv.ssd.GetIfNewer(name, version)
	if !ok {
		http.NotFound(writer, request)
		return
	}

	source, err := os.Open(filepath.Join(srv.config.HostsDir(), name))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer source.Close()

	writer.Header().Set("X-Name", info.Name)
	writer.Header().Set("X-Version", strconv.FormatInt(info.Version, 10))
	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, source)
}
