package scripts

import (
	"bytes"
	"text/template"
)

var Assembly = template.Must(template.New("").Parse(`#!/usr/bin/env bash
set -e -o pipefail -x

BIN_URL="${BIN_URL:-https://github.com/reddec/tinc-boot/releases/latest/download/tinc-boot_linux_{{.Platform}}.tar.gz}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
BIN="$BIN_DIR/tinc-boot"
NETWORK="${NETWORK:-{{.Network}}}"
PORT="${PORT:-{{.Port}}}"
MASK="${MASK:-{{.Mask}}}"
ADDRESS="${ADDRESS:-{{.Address}}}"
NAME="${NAME:-{{.Name}}}"
ROOT="/etc/tinc/$NETWORK"

if [[ $(/usr/bin/id -u) -ne 0 ]]; then
	echo "the script should be run under root/sudo"
	exit 1
fi


function apt_based {
	apt update && apt install -y tinc curl	
}

if command -v apt; then
	apt_based	
fi

if ! command -v tincd; then
	echo "tinc not installed"
	exit 1
fi

if ! command -v "$BIN"; then
	if ! command -v curl; then
		echo "no installed curl"
		exit 1
	fi
    echo "Installing tinc-boot to $BIN_DIR"
    curl -L "$BIN_URL" | tar -xz -C "$BIN_DIR" tinc-boot
    chmod +x "$BIN"
fi

"$BIN" gen\
  --name "$NAME" \
  --network "$NETWORK" --bin "$BIN" \
  --no-gen-key \
  --no-bin-copy \
  --bin "$BIN" \
  --port "$PORT" \
  --mask "$MASK" \
  --standalone \
  {{range $name, $file := .ConnectTo}}--connect-to "{{$name}}" {{end}}{{with .ConnectTo}}\{{end}}
  --prefix "$ADDRESS"{{with .Public}}\{{end}}
  {{range .Public}}--public "{{.}}" {{end}}

cat - >> "$ROOT/hosts/${NAME}" <<EOF
{{.HostPublic}}
EOF

cat - > "$ROOT/rsa_key.priv" <<EOF
{{.HostPrivate}}
EOF

chmod u=rw "$ROOT/rsa_key.priv"
{{range $name, $file := .ConnectTo}}
cat - >> "$ROOT/hosts/{{$name}}" <<EOF
{{$file}}
EOF
{{end}}

mkdir /dev/net
mknod /dev/net/tun c 10 200

systemctl enable tinc@${NETWORK}
systemctl start tinc@${NETWORK}
`))

type AssemblyParam struct {
	Public      []string
	Platform    string
	Name        string
	Network     string
	Address     string
	Mask        int
	Port        int
	ConnectTo   map[string]string
	HostPublic  string
	HostPrivate string
}

func (cfg *AssemblyParam) Render() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := Assembly.Execute(buf, *cfg)
	return buf.Bytes(), err
}
