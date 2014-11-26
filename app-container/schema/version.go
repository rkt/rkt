package schema

import (
	"github.com/coreos-inc/rkt/app-container/schema/types"
)

var (
	AppContainerVersion types.SemVer
)

func init() {
	v, _ := types.NewSemVer("0.1.0")
	AppContainerVersion = *v
}
