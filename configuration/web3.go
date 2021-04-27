package configuration

import "crypto/ecdsa"

type Web3 struct {
	URL           string            `yaml:"url"`
	RawPrivateKey string            `yaml:"private_key"`
	PrivateKey    *ecdsa.PrivateKey `yaml:"-"`
}
