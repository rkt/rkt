package invoke

import (
	"fmt"
	"os"
	"strings"

	"github.com/appc/cni/pkg/types"
)

func DelegateAdd(delegatePlugin string, netconf []byte) (*types.Result, error) {
	if os.Getenv("CNI_COMMAND") != "ADD" {
		return nil, fmt.Errorf("CNI_COMMAND is not ADD")
	}

	paths := strings.Split(os.Getenv("CNI_PATH"), ":")

	pluginPath, err := FindInPath(delegatePlugin, paths)
	if err != nil {
		return nil, err
	}

	return ExecPluginWithResult(pluginPath, netconf, ArgsFromEnv())
}

func DelegateDel(delegatePlugin string, netconf []byte) error {
	if os.Getenv("CNI_COMMAND") != "DEL" {
		return fmt.Errorf("CNI_COMMAND is not DEL")
	}

	paths := strings.Split(os.Getenv("CNI_PATH"), ":")

	pluginPath, err := FindInPath(delegatePlugin, paths)
	if err != nil {
		return err
	}

	return ExecPluginWithoutResult(pluginPath, netconf, ArgsFromEnv())
}
