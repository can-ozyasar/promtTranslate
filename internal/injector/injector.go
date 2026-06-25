package injector

import "context"

// Injector types text into the currently active window.
type Injector interface {
	// Type simulates typing text into the active window.
	Type(ctx context.Context, text string) error
}
