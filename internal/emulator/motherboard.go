package emulator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrInvalidFrequency = errors.New("motherboard: frequency must be > 0")

type Config struct {
	FrequencyHz uint64
}

// Motherboard orchestrates components and drives cycles from the clock.
type Motherboard struct {
	mu         sync.RWMutex
	frequency  uint64
	bus        *Bus
	components []ClockedComponent
	cycle      uint64
}

func NewMotherboard(cfg Config) (*Motherboard, error) {
	if cfg.FrequencyHz == 0 {
		return nil, ErrInvalidFrequency
	}

	return &Motherboard{
		frequency: cfg.FrequencyHz,
		bus:       NewBus(),
	}, nil
}

func (m *Motherboard) Bus() *Bus {
	return m.bus
}

func (m *Motherboard) AddComponent(c ClockedComponent) error {
	if c == nil {
		return errors.New("motherboard: component is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.components = append(m.components, c)
	return nil
}

func (m *Motherboard) Reset(ctx context.Context) error {
	components := m.snapshotComponents()
	for _, c := range components {
		if err := c.Reset(ctx); err != nil {
			return fmt.Errorf("motherboard: reset %s: %w", c.Name(), err)
		}
	}

	m.mu.Lock()
	m.cycle = 0
	m.mu.Unlock()

	return nil
}

func (m *Motherboard) Close() error {
	components := m.snapshotComponents()
	var errs []error

	for _, c := range components {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name(), err))
		}
	}

	return errors.Join(errs...)
}

func (m *Motherboard) Cycle() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cycle
}

func (m *Motherboard) Run(ctx context.Context) error {
	period := time.Second / time.Duration(m.frequency)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.Step(ctx); err != nil {
				return err
			}
		}
	}
}

func (m *Motherboard) Step(ctx context.Context) error {
	components := m.snapshotComponents()
	cycle := m.Cycle()
	tick := Tick{Cycle: cycle}

	errCh := make(chan error, len(components))
	var wg sync.WaitGroup

	for _, c := range components {
		comp := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := comp.Tick(ctx, tick, m.bus); err != nil {
				errCh <- fmt.Errorf("%s: %w", comp.Name(), err)
			}
		}()
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	m.mu.Lock()
	m.cycle++
	m.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("motherboard: tick cycle %d failed: %w", cycle, errors.Join(errs...))
	}

	return nil
}

func (m *Motherboard) snapshotComponents() []ClockedComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]ClockedComponent, len(m.components))
	copy(out, m.components)
	return out
}
