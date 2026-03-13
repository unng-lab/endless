package rl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return LinearQStubArtifact{}, fmt.Errorf("linear q stub artifact path %q points to a directory, want one JSON artifact file written by train-stub", path)
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		return LinearQStubArtifact{}, fmt.Errorf("read linear q stub artifact %q: %w", path, err)
	}

	var artifact LinearQStubArtifact
	if err := json.Unmarshal(payload, &artifact); err != nil {
		if formatErr := detectUnsupportedRuntimeArtifact(path, payload); formatErr != nil {
			return LinearQStubArtifact{}, formatErr
		}
		return LinearQStubArtifact{}, fmt.Errorf("unmarshal linear q stub artifact %q: %w", path, err)
	}
	if err := artifact.Validate(); err != nil {
		if formatErr := detectUnsupportedRuntimeArtifact(path, payload); formatErr != nil {
			return LinearQStubArtifact{}, formatErr
		}
		return LinearQStubArtifact{}, err
	}
	return artifact, nil
}

// detectUnsupportedRuntimeArtifact recognizes trainer-side GoMLX outputs so runtime callers get
// one actionable error instead of a misleading tensor-dimension mismatch from the stub loader.
func detectUnsupportedRuntimeArtifact(path string, payload []byte) error {
	lowerPath := strings.ToLower(path)
	if strings.HasSuffix(lowerPath, ".bin") {
		return fmt.Errorf("file %q looks like a GoMLX checkpoint binary, not a linear q stub runtime artifact; -rl-model-path only supports JSON artifacts written by cmd/endless-rl-train -mode train-stub -train-model-output", path)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil
	}

	if hasJSONKeys(root, "trained_at", "obs_dim", "action_dim", "input_dim", "checkpoint_dir") {
		return fmt.Errorf("file %q is a GoMLX critic trainer manifest, not a linear q stub runtime artifact; use LoadRuntimePolicyFromPath or pass this path through -rl-model-path instead of the linear-q-stub-only loader", path)
	}
	if hasJSONKeys(root, "Variables", "BinFormat") {
		return fmt.Errorf("file %q is GoMLX checkpoint metadata, not a linear q stub runtime artifact; use LoadRuntimePolicyFromPath or pass this path through -rl-model-path instead of the linear-q-stub-only loader", path)
	}
	return nil
}

// hasJSONKeys keeps the format detection explicit and readable because the loader only needs a
// small set of stable top-level markers to distinguish trainer artifacts from runtime ones.
func hasJSONKeys(root map[string]json.RawMessage, keys ...string) bool {
	for _, key := range keys {
		if _, ok := root[key]; !ok {
			return false
		}
	}
	return true
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
