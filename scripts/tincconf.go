package scripts

import (
	"bytes"
	"text/template"
)

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
