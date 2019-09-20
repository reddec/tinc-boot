package scripts

import (
	"bytes"
	"text/template"
)

var TincUp = template.Must(template.New("").Parse(`#!/usr/bin/env bash
ip addr add {{.Addr}}/{{.Mask}} dev $INTERFACE
ip link set dev $INTERFACE up
{{.Bin}} monitor &
`))

type TincUpParam struct {
	Addr string
	Mask int
	Bin  string
}

func (cfg *TincUpParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := TincUp.Execute(buf, *cfg)
	return buf.Bytes(), err
}
