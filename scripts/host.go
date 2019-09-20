package scripts

import (
	"bytes"
	"text/template"
)

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
