package scripts

import (
	"bytes"
	"text/template"
)

var TincDown = template.Must(template.New("").Parse(`#!/usr/bin/env bash
{{.Bin}} kill
ip addr del {{.Addr}}/{{.Mask}} dev $INTERFACE
ip link set dev $INTERFACE down
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
