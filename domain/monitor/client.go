package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/reddec/tinc-boot/types"
	"io/ioutil"
	"net/http"
	"strings"
)

type Client interface {
	GetHostFile() (string, error)
	Watch(subnet types.Subnet) error
	Forget(subnetAddr string) error
	Kill() error
	Nodes() (*NodeList, error)
	GetNodeFile(node string) (string, error)
	PushNodeFile(node string, content string) error
}

func ClientBySubnet(subnet string) Client {
	addr := strings.SplitN(subnet, "/", 2)[0]
	return &httpAPI{baseURL: "http://" + addr + ":1655"}
}

type httpAPI struct {
	baseURL string
}

func (api *httpAPI) GetHostFile() (string, error) {
	return api.requestString("/")
}

func (api *httpAPI) Watch(subnet types.Subnet) error {
	data, err := json.Marshal(subnet)
	if err != nil {
		return err
	}
	return api.pushRequest("/rpc/watch", data, "application/json")
}

func (api *httpAPI) Forget(subnetAddr string) error {
	data, err := json.Marshal(types.Subnet{
		Subnet: subnetAddr,
	})
	if err != nil {
		return err
	}
	return api.pushRequest("/rpc/forget", data, "application/json")
}

func (api *httpAPI) Kill() error {
	return api.pushRequest("/rpc/kill", nil, "text/plain")
}

func (api *httpAPI) Nodes() (*NodeList, error) {
	res, err := api.requestString("/rpc/nodes")
	if err != nil {
		return nil, err
	}
	var ans NodeList
	return &ans, json.Unmarshal([]byte(res), &ans)
}

func (api *httpAPI) GetNodeFile(node string) (string, error) {
	return api.requestString("/rpc/node/" + node + "/hostfile")
}

func (api *httpAPI) PushNodeFile(node string, content string) error {
	return api.pushRequest("/rpc/node/"+node+"/hostfile", []byte(content), "text/plain")
}

func (api *httpAPI) requestString(path string) (string, error) {
	res, err := http.Get(api.baseURL + path)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%v: %v: %v", res.StatusCode, res.Status, string(data))
	}
	return string(data), nil
}

func (api *httpAPI) pushRequest(path string, data []byte, contentType string) error {
	res, err := http.Post(api.baseURL+path, contentType, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	rdata, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%v: %v: %v", res.StatusCode, res.Status, string(rdata))
	}
	return nil
}
