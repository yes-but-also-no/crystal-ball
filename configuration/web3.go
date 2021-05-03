package configuration

import "crypto/ecdsa"

type Web3 struct {
	URL           string            `yaml:"url"`
	RawPrivateKey string            `yaml:"private_key"`
	OrakuruCore   string            `yaml:"orakuru_core"`
	PrivateKey    *ecdsa.PrivateKey `yaml:"-"`
}
