package core

// ResetHooks empties every hook and clears the emitted flags, the installing
// feature, the installed features and Trace, so each test starts from a clean kernel.
func ResetHooks() {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	for _, h := range hooks {
		h.reset()
	}
	installing = ""
	Trace = nil
	resetFeatures()
}

// SetInstalling sets the feature name Install would set.
func SetInstalling(feature string) { installing = feature }
