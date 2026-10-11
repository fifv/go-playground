//go:build !windows

package main

import (
	"context"
	"errors"
)

func play(context.Context, options) error {
	return errors.New("audio playback currently requires Windows")
}
