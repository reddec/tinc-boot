package scripts

import (
	"bytes"
	"net"
	"text/template"
)

const Extension = ".bat"

var TincUp = template.Must(template.New("").Parse(`
netsh interface ipv4 set address name="%INTERFACE%" static {{.Addr}} {{.MaskAsAddr}} store=persistent
cd "%~dp0"
start /B "" "{{.Bin}}" monitor
`))

type TincUpParam struct {
	Addr string
	Mask int
	Bin  string
}

func (tup TincUpParam) MaskAsAddr() string { return maskIpV4AsSubnet(tup.Mask) }

func (cfg *TincUpParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := TincUp.Execute(buf, *cfg)
	return buf.Bytes(), err
}

var TincDown = template.Must(template.New("").Parse(`
"{{.Bin}}" kill
`))

type TincDownParam struct {
	Bin  string
	Addr string
	Mask int
}

func (cfg *TincDownParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := TincDown.Execute(buf, *cfg)
	return buf.Bytes(), err
}

var TincConf = template.Must(template.New("").Parse(`
Name = {{.Name}}
LocalDiscovery = yes
Interface = {{.Net}}
Port = {{.Port}}
{{range .ConnectTo}}
ConnectTo = {{.}}
{{end}}
`))

type TincConfParam struct {
	Name      string
	Net       string
	Port      int
	ConnectTo []string
}

func (cfg *TincConfParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := TincConf.Execute(buf, *cfg)
	return buf.Bytes(), err
}

var SubnetUp = template.Must(template.New("").Parse(`
"{{.Bin}}" watch
`))

type SubnetUpParam struct {
	Bin string
}

func (cfg *SubnetUpParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := SubnetUp.Execute(buf, *cfg)
	return buf.Bytes(), err
}

var SubnetDown = template.Must(template.New("").Parse(`
"{{.Bin}}" forget
`))

type SubnetDownParam struct {
	Bin string
}

func (cfg *SubnetDownParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := SubnetDown.Execute(buf, *cfg)
	return buf.Bytes(), err
}

var Host = template.Must(template.New("").Parse(`
Subnet = {{.Address}}/32
{{- range .Public}}
Address = {{.}}
{{- end}}
Port = {{.Port}}
`))

type HostParam struct {
	Public  []string
	Address string
	Port    int
}

func (cfg *HostParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := Host.Execute(buf, *cfg)
	return buf.Bytes(), err
}

func maskIpV4AsSubnet(bits int) string {
	mask := net.CIDRMask(16, 8*net.IPv4len)
	return net.IPv4(mask[0], mask[1], mask[2], mask[3]).String()
}
