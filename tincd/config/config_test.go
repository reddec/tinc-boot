package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {

	type config struct {
		Name      string `tinc:"name"`
		Port      uint16
		PublicKey []byte `tinc:"RSA PUBLIC KEY"`
	}
	var cfg config
	err := Unmarshal([]byte(`
# comment
Address = paas.reddec.net 1655
Name = paasreddecnet_5TA7JX
Port = 1655
Subnet = 10.155.0.0/16
Version = 2

-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEAx3+0Uvin/9z2V6JD/m3VSRJvCjc6ecYJS1o1ThTaIFTWxPylfDdO
nCpWFu3/Z9IHSyHGce7pqm+h9lUDR9scSTelMuz+w7VDb5zSHooFDG765mFjMA4s
z/o4SM8oadTBM3UtRd6d4D4ZaBBGA5RbH4k92aYGhwqIkI+goj0buVNdsi4kPlZP
SX0Cma5OXgKGihSutSSdIcbu4f5iYovKFPpLuz3I3oSegbqfgphe24vkw0HOSmrQ
SD8YLkwGd9azx0/087/FLqXJGo0b+0pAArkxIXiVORJbaiTZ18piMwG3/sQnTibY
hGXCTofVokYwaAYzKdAwHu7CRqj+/HaX+LWYzQa0Mw+G6j/VMgwFv0slZVI7jLXy
3RwsZgdqc6G5bswIPvgADbTQKt4obo82An7WwAJq5rHkrNrR2AcEE3m8CfXw3B6c
ftEDqnq/uDgH1tJp/k5NLeuZkV1VrCS6a07+vkQLit//7wL55StZu7wxmByceYfI
qV4nF62kVF6Uo613WFI4P3y82Pi4YNT7lG6oqUg1IAbYej7TBCM8HfWWWvNoXyTa
POLXf3pbVgxjXp35BvJ8pZAqw96FeDTCnr+FOVNtOrMLO2BCTO68ciHj2uxJb3f6
vKrHmqOR+xd3UBpr7QCTSjAzi7TtGi5KF/qXd7PVQZZfA8soyjwyWN8CAwEAAQ==
-----END RSA PUBLIC KEY-----
`), &cfg)
	if !assert.NoError(t, err, "parse") {
		return
	}
	assert.Equal(t, "paasreddecnet_5TA7JX", cfg.Name)
	assert.Equal(t, uint16(1655), cfg.Port)
	t.Logf("%+v", cfg)
}

func TestMarshal(t *testing.T) {
	type config struct {
		Name      string `tinc:"name"`
		Port      uint16
		PublicKey []byte `tinc:"RSA PUBLIC KEY,blob"`
	}
	var cfg = config{
		Name:      "XXXX",
		Port:      1456,
		PublicKey: []byte("----BEGIN RSA PUBLIC KEY ----\n1\n2\n3\n----END RSA PUBLIC KEY ----"),
	}
	data, err := Marshal(cfg)
	assert.NoError(t, err)
	t.Log(string(data))
}
