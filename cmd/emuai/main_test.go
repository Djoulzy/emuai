package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunControlTogglePaused(t *testing.T) {
	control := &runControl{}

	if control.Paused() {
		t.Fatal("expected control to start unpaused")
	}

	if !control.TogglePaused() {
		t.Fatal("expected first toggle to pause execution")
	}

	if !control.Paused() {
		t.Fatal("expected paused state after first toggle")
	}

	if control.TogglePaused() {
		t.Fatal("expected second toggle to resume execution")
	}

	if control.Paused() {
		t.Fatal("expected resumed state after second toggle")
	}
}

func TestWaitWhilePausedReturnsWhenResumed(t *testing.T) {
	control := &runControl{}
	control.SetPaused(true)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(pausePollInterval / 2)
		control.SetPaused(false)
	}()

	if err := waitWhilePaused(ctx, control); err != nil {
		t.Fatalf("waitWhilePaused returned error: %v", err)
	}
}

func TestWaitWhilePausedStopsOnContextCancellation(t *testing.T) {
	control := &runControl{}
	control.SetPaused(true)

	ctx, cancel := context.WithTimeout(context.Background(), pausePollInterval)
	defer cancel()

	err := waitWhilePaused(ctx, control)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestProcessControlKeyPauseResume(t *testing.T) {
	control := &runControl{}

	if action := processControlKey(control, nil, ' '); action != "pause" {
		t.Fatalf("expected pause action, got %q", action)
	}

	if !control.Paused() {
		t.Fatal("expected paused state after space")
	}

	if action := processControlKey(control, nil, ' '); action != "resume" {
		t.Fatalf("expected resume action, got %q", action)
	}

	if control.Paused() {
		t.Fatal("expected resumed state after second space")
	}
}

func TestProcessControlKeyQuit(t *testing.T) {
	called := false
	action := processControlKey(&runControl{}, func() {
		called = true
	}, 'q')

	if action != "quit" {
		t.Fatalf("expected quit action, got %q", action)
	}

	if !called {
		t.Fatal("expected quit callback to be called")
	}
}
