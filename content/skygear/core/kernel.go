package core

// KernelFeature is the kernel's own feature (the shapes and parts steps of
// the lifecycle); the game lists it first.
var KernelFeature = &Feature{Name: "core"}

// AutofillFeature is the autofill feature (the refill of a used-up hand
// stack); the game lists it after the features whose steps it follows.
var AutofillFeature = &Feature{Name: "autofill"}
