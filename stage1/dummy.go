package dummy

// This is a dummy package to ensure that Godep vendors
// actool, which is used in building the stage1 ACI
import (
	_ "github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/aci"
	_ "github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/actool"
)
