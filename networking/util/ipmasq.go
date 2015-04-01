package util

import (
	"fmt"
	"net"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/coreos/go-iptables/iptables"
)

// Installs iptables rules to masquerade traffic
// coming from ipn and going outside of it
func SetupIPMasq(ipn *net.IPNet, chain string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to locate iptabes: %v", err)
	}

	if err = ipt.NewChain("nat", chain); err != nil {
		if err.(*iptables.Error).ExitStatus() != 1 {
			// TODO(eyakubovich): assumes exit status 1 implies chain exists
			return err
		}
	}

	if err = ipt.AppendUnique("nat", chain, "-d", ipn.String(), "-j", "ACCEPT"); err != nil {
		return err
	}

	if err = ipt.AppendUnique("nat", chain, "!", "-d", "224.0.0.0/4", "-j", "MASQUERADE"); err != nil {
		return err
	}

	return ipt.AppendUnique("nat", "POSTROUTING", "-s", ipn.String(), "-j", chain)
}

// Undoes the effects of SetupIPMasq
func TeardownIPMasq(ipn *net.IPNet, chain string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to locate iptabes: %v", err)
	}

	if err = ipt.Delete("nat", "POSTROUTING", "-s", ipn.String(), "-j", chain); err != nil {
		return err
	}

	if err = ipt.ClearChain("nat", chain); err != nil {
		return err
	}

	return ipt.DeleteChain("nat", chain)
}
