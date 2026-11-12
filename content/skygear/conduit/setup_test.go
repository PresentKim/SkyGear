package conduit_test

import (
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/network"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once. The network feature runs the energy, item and
// fluid step the conduit tests wait for.
func init() { coretest.Setup(conduit.Feature, network.Feature) }
