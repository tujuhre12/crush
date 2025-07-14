package notification

import (
	"context"
	"testing"
	"time"
)

func TestNotifier(t *testing.T) {
	// Test with notifications enabled
	notifier := New(true)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This should not panic or error
	notifier.NotifyTaskComplete(ctx, "Test Title", "Test message")

	// Test with notifications disabled
	disabledNotifier := New(false)
	disabledNotifier.NotifyTaskComplete(ctx, "Test Title", "Test message")
}

func TestNotifierDisabled(t *testing.T) {
	notifier := New(false)
	ctx := context.Background()

	// Should return immediately without doing anything
	notifier.NotifyTaskComplete(ctx, "Test Title", "Test message")
}
