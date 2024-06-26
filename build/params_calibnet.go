//go:build calibnet
// +build calibnet

package build

import "github.com/filecoin-project/go-address"

func init() {
	SetAddressNetwork(address.Testnet)
	BuildType = "+calibnet"
}

func SetAddressNetwork(n address.Network) {
	address.CurrentNetwork = n
}
