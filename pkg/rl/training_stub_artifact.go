package rl

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const linearQStubArtifactVersion = 1

// LinearQStubArtifact keeps the exact tensor contract together with one trained linear model so
// offline smoke-check training may hand the resulting weights to runtime or external inspection.
type LinearQStubArtifact struct {
	Version           int                         `json:"version"`
	SavedAt           time.Time                   `json:"saved_at"`
	NormalizationSpec TransitionNormalizationSpec `json:"normalization_spec"`
	Model             LinearQStubModel            `json:"model"`
}

// SaveLinearQStubArtifact writes one trained stub model together with the frozen normalization
// spec that defines the observation and action layout expected by the runtime scorer.
func SaveLinearQStubArtifact(path string, spec TransitionNormalizationSpec, model LinearQStubModel) error {
	if path == "" {
		return fmt.Errorf("artifact path is empty")
	}

	artifact := LinearQStubArtifact{
		Version:           linearQStubArtifactVersion,
		SavedAt:           time.Now().UTC(),
		NormalizationSpec: spec.Normalized(),
		Model:             model,
	}
	if err := artifact.Validate(); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal linear q stub artifact: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write linear q stub artifact %q: %w", path, err)
	}
	return nil
}

// LoadLinearQStubArtifact restores one serialized model file and re-validates its tensor layout
// before runtime code starts scoring actions against live observations.
func LoadLinearQStubArtifact(path string) (LinearQStubArtifact, error) {
	if path == "" {
		return LinearQStubArtifact{}, fmt.Errorf("artifact path is empty")
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		return LinearQStubArtifact{}, fmt.Errorf("read linear q stub artifact %q: %w", path, err)
	}

	var artifact LinearQStubArtifact
	if err := json.Unmarshal(payload, &artifact); err != nil {
		return LinearQStubArtifact{}, fmt.Errorf("unmarshal linear q stub artifact %q: %w", path, err)
	}
	if err := artifact.Validate(); err != nil {
		return LinearQStubArtifact{}, err
	}
	return artifact, nil
}

// Validate checks that the serialized model and normalization spec still describe one coherent
// tensor contract before any caller relies on them for scoring or gameplay inference.
func (a LinearQStubArtifact) Validate() error {
	if a.Version != linearQStubArtifactVersion {
		return fmt.Errorf("linear q stub artifact version = %d, want %d", a.Version, linearQStubArtifactVersion)
	}

	spec := a.NormalizationSpec.Normalized()
	if a.Model.ObsDim != spec.ObservationDim() {
		return fmt.Errorf("linear q stub artifact observation dim = %d, want %d", a.Model.ObsDim, spec.ObservationDim())
	}
	if a.Model.ActionDim != spec.ActionDim() {
		return fmt.Errorf("linear q stub artifact action dim = %d, want %d", a.Model.ActionDim, spec.ActionDim())
	}

	expectedWeights := a.Model.ObsDim + a.Model.ActionDim
	if len(a.Model.Weights) != expectedWeights {
		return fmt.Errorf("linear q stub artifact weights = %d, want %d", len(a.Model.Weights), expectedWeights)
	}
	return nil
}
