package storagenet

import "errors"

// Errors returned by the storage network engine.
var (
	// ErrCycle reports that a crafting plan would depend on itself.
	ErrCycle = errors.New("storagenet: crafting cycle")
	// ErrBudget reports that a crafting plan exceeded its step budget.
	ErrBudget = errors.New("storagenet: crafting budget exceeded")
	// ErrNoPattern reports that no pattern produces a wanted key.
	ErrNoPattern = errors.New("storagenet: no pattern produces key")
	// ErrShort reports that the storage does not hold enough of a key.
	ErrShort = errors.New("storagenet: not enough items in storage")
)

// Key identifies an item kind stored in a network.
type Key string

// Stack is a count of one key.
type Stack struct {
	// Key identifies the item kind.
	Key Key
	// Count is the number of items.
	Count int64
}

// Storage holds item counts by key.
type Storage interface {
	// Insert adds up to n items of key k and returns how many were added.
	// When sim is true nothing is changed and the result is what would be added.
	Insert(k Key, n int64, sim bool) int64
	// Extract removes up to n items of key k and returns how many were removed.
	// When sim is true nothing is changed and the result is what would be removed.
	Extract(k Key, n int64, sim bool) int64
	// Count returns the number of items of key k.
	Count(k Key) int64
	// Each calls fn once for every key with its count.
	Each(fn func(Key, int64))
}

// Ingredient is one input slot of a pattern.
type Ingredient struct {
	// Count is the number of items the slot consumes per craft.
	Count int64
	// Options lists the keys that may fill the slot.
	Options []Key
}

// Pattern describes a craft that turns inputs into outputs.
type Pattern struct {
	// ID uniquely identifies the pattern.
	ID string
	// Processing reports whether the pattern is a processing recipe.
	Processing bool
	// Inputs are the ingredients consumed per craft.
	Inputs []Ingredient
	// Outputs are the stacks produced per craft.
	Outputs []Stack
}

// Step is one entry of a crafting plan.
type Step struct {
	// PatternID names the pattern to run.
	PatternID string
	// Times is how many times the pattern runs.
	Times int64
}

// Plan is the result of resolving a wanted stack into crafting steps.
type Plan struct {
	// Want is the stack the plan was built for.
	Want Stack
	// Steps are the pattern runs needed to make Want.
	Steps []Step
	// Use counts the items the plan takes from storage, by key.
	Use map[Key]int64
	// Missing counts the items the plan still lacks, by key.
	Missing map[Key]int64
}

// OK reports whether the plan has nothing missing.
func (p *Plan) OK() bool {
	return len(p.Missing) == 0
}
