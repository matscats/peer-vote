package ports

import "time"

// Clock is a port interface for time operations
// This abstraction allows for testable time control in the consensus engine
// In production, use RealClock; in tests, use a mock clock
type Clock interface {
	// Now returns the current time
	Now() time.Time

	// NewTicker creates a new ticker that ticks at the specified interval
	NewTicker(interval time.Duration) Ticker
}

// Ticker is an interface for time.Ticker
// This allows for testable ticker behavior
type Ticker interface {
	// C returns the channel on which ticks are delivered
	C() <-chan time.Time

	// Stop stops the ticker
	Stop()

	// Reset resets the ticker to a new interval
	Reset(interval time.Duration)
}

// RealClock implements Clock using the standard time package
// This is the production implementation that uses actual system time
type RealClock struct{}

// NewRealClock creates a new RealClock
func NewRealClock() *RealClock {
	return &RealClock{}
}

// Now returns the current time
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// NewTicker creates a new ticker
func (c *RealClock) NewTicker(interval time.Duration) Ticker {
	return &realTicker{
		ticker: time.NewTicker(interval),
	}
}

// realTicker wraps time.Ticker to implement the Ticker interface
type realTicker struct {
	ticker *time.Ticker
}

// C returns the ticker channel
func (t *realTicker) C() <-chan time.Time {
	return t.ticker.C
}

// Stop stops the ticker
func (t *realTicker) Stop() {
	t.ticker.Stop()
}

// Reset resets the ticker
func (t *realTicker) Reset(interval time.Duration) {
	t.ticker.Reset(interval)
}
