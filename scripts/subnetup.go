package scripts

import (
	"bytes"
	"text/template"
)

var SubnetUp = template.Must(template.New("").Parse(`#!/usr/bin/env bash
{{.Bin}} watch
`))

type SubnetUpParam struct {
	Bin string
}

func (cfg *SubnetUpParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := SubnetUp.Execute(buf, *cfg)
	return buf.Bytes(), err
}
