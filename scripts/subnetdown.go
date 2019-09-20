package scripts

import (
	"bytes"
	"text/template"
)

var SubnetDown = template.Must(template.New("").Parse(`#!/usr/bin/env bash
{{.Bin}} forget
`))

type SubnetDownParam struct {
	Bin string
}

func (cfg *SubnetDownParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := SubnetDown.Execute(buf, *cfg)
	return buf.Bytes(), err
}
