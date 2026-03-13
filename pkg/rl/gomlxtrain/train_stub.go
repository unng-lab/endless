//go:build !linux

package gomlxtrain

import (
	"context"
	"fmt"
)

// TrainCritic is only available on Linux because the intended GoMLX/XLA + WSL2 execution path
// depends on Linux-side PJRT plugins and CUDA tooling.
func TrainCritic(_ context.Context, _ Config) (Result, error) {
	return Result{}, fmt.Errorf("endless GoMLX trainer requires Linux/WSL2; run this command inside WSL2")
}
