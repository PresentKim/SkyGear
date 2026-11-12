package expand_test

import (
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/expand"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once, with the kernel's own feature.
func init() { coretest.Setup(core.KernelFeature, expand.Feature) }
