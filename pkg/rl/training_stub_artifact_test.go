package rl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLinearQStubArtifactRoundTrip(t *testing.T) {
	spec := DefaultTransitionNormalizationSpec()
	model, err := NewLinearQStubModel(spec.ObservationDim(), spec.ActionDim())
	if err != nil {
		t.Fatalf("NewLinearQStubModel() error = %v", err)
	}
	model.Bias = 1.25
	if len(model.Weights) > 0 {
		model.Weights[0] = 0.5
	}

	path := filepath.Join(t.TempDir(), "linear_q_stub_artifact.json")
	if err := SaveLinearQStubArtifact(path, spec, model); err != nil {
		t.Fatalf("SaveLinearQStubArtifact() error = %v", err)
	}

	artifact, err := LoadLinearQStubArtifact(path)
	if err != nil {
		t.Fatalf("LoadLinearQStubArtifact() error = %v", err)
	}
	if got, want := artifact.Model.ObsDim, spec.ObservationDim(); got != want {
		t.Fatalf("artifact.Model.ObsDim = %d, want %d", got, want)
	}
	if got, want := artifact.Model.ActionDim, spec.ActionDim(); got != want {
		t.Fatalf("artifact.Model.ActionDim = %d, want %d", got, want)
	}
	if got, want := artifact.Model.Bias, float32(1.25); got != want {
		t.Fatalf("artifact.Model.Bias = %f, want %f", got, want)
	}
	if got, want := artifact.Model.Weights[0], float32(0.5); got != want {
		t.Fatalf("artifact.Model.Weights[0] = %f, want %f", got, want)
	}
}

func TestLoadLinearQStubArtifactRejectsGoMLXTrainerManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gomlx_critic_manifest.json")
	payload := []byte(`{
  "version": 1,
  "trained_at": "2026-03-12T21:31:28Z",
  "obs_dim": 355,
  "action_dim": 8,
  "input_dim": 363,
  "checkpoint_dir": "./artifacts/gomlx_critic_cover"
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadLinearQStubArtifact(path)
	if err == nil {
		t.Fatal("LoadLinearQStubArtifact() error = nil, want trainer manifest rejection")
	}
	if !strings.Contains(err.Error(), "GoMLX critic trainer manifest") {
		t.Fatalf("LoadLinearQStubArtifact() error = %q, want GoMLX trainer manifest hint", err)
	}
}

func TestLoadLinearQStubArtifactRejectsGoMLXCheckpointMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint-n0000020-step-00064080.json")
	payload := []byte(`{
  "Variables": [
    {
      "ParameterName": "var:/hidden_0/dense/weights",
      "Dimensions": [363, 256],
      "DType": "Float32"
    }
  ],
  "BinFormat": "gzip"
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadLinearQStubArtifact(path)
	if err == nil {
		t.Fatal("LoadLinearQStubArtifact() error = nil, want checkpoint metadata rejection")
	}
	if !strings.Contains(err.Error(), "GoMLX checkpoint metadata") {
		t.Fatalf("LoadLinearQStubArtifact() error = %q, want GoMLX checkpoint metadata hint", err)
	}
}
