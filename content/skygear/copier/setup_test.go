package copier_test

import (
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
)

// This package's test binary links no feature list, so no feature is
// installed in it. The tests that reach a feature through the kernel's hooks
// have it set up here, once.
func init() { coretest.Setup(copier.Feature) }
