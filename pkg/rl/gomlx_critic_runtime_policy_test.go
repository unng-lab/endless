package rl

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestGoMLXCriticRuntimeModelPredictAppliesDenseReluDense(t *testing.T) {
	model := goMLXCriticRuntimeModel{
		InputDim: 2,
		HiddenLayers: []goMLXCriticDenseLayer{
			{
				InDim:   2,
				OutDim:  2,
				Weights: []float32{1, -1, 2, 0},
				Biases:  []float32{0.5, -0.5},
			},
		},
		OutputLayer: goMLXCriticDenseLayer{
			InDim:   2,
			OutDim:  1,
			Weights: []float32{2, 3},
			Biases:  []float32{1},
		},
	}

	prediction, err := model.Predict([]float32{1, 2})
	if err != nil {
		t.Fatalf("Predict() error = %v", err)
	}
	if got, want := prediction, float32(12); got != want {
		t.Fatalf("Predict() = %f, want %f", got, want)
	}
}

func TestLoadGoMLXCriticRuntimePolicyFromManifestUsesLatestCheckpoint(t *testing.T) {
	dir := t.TempDir()
	manifest := testGoMLXCriticManifest(dir, []int{2})
	if err := writeTestGoMLXCriticManifest(filepath.Join(dir, goMLXCriticManifestFileName), manifest); err != nil {
		t.Fatalf("writeTestGoMLXCriticManifest() error = %v", err)
	}

	oldBase := "checkpoint-n0000019-20260313-003128-step-00064080"
	newBase := "checkpoint-n0000020-20260313-003128-step-00064080"
	if err := writeZeroedTestGoMLXCheckpoint(filepath.Join(dir, oldBase), manifest.InputDim, manifest.HiddenDims, 1); err != nil {
		t.Fatalf("writeZeroedTestGoMLXCheckpoint(old) error = %v", err)
	}
	if err := writeZeroedTestGoMLXCheckpoint(filepath.Join(dir, newBase), manifest.InputDim, manifest.HiddenDims, 2); err != nil {
		t.Fatalf("writeZeroedTestGoMLXCheckpoint(new) error = %v", err)
	}

	policy, err := LoadGoMLXCriticRuntimePolicy(filepath.Join(dir, goMLXCriticManifestFileName))
	if err != nil {
		t.Fatalf("LoadGoMLXCriticRuntimePolicy() error = %v", err)
	}
	if got, want := policy.artifact.CheckpointJSONPath, filepath.Join(dir, newBase+".json"); got != want {
		t.Fatalf("policy.artifact.CheckpointJSONPath = %q, want %q", got, want)
	}

	input := make([]float32, manifest.InputDim)
	prediction, err := policy.artifact.Model.Predict(input)
	if err != nil {
		t.Fatalf("policy.artifact.Model.Predict() error = %v", err)
	}
	if got, want := prediction, float32(2); got != want {
		t.Fatalf("policy.artifact.Model.Predict() = %f, want %f", got, want)
	}
}

func TestLoadRuntimePolicyFromPathDetectsGoMLXCritic(t *testing.T) {
	dir := t.TempDir()
	manifest := testGoMLXCriticManifest(dir, []int{2})
	if err := writeTestGoMLXCriticManifest(filepath.Join(dir, goMLXCriticManifestFileName), manifest); err != nil {
		t.Fatalf("writeTestGoMLXCriticManifest() error = %v", err)
	}
	if err := writeZeroedTestGoMLXCheckpoint(filepath.Join(dir, "checkpoint-n0000001-20260313-003128-step-00000010"), manifest.InputDim, manifest.HiddenDims, 3); err != nil {
		t.Fatalf("writeZeroedTestGoMLXCheckpoint() error = %v", err)
	}

	policy, label, err := LoadRuntimePolicyFromPath(dir)
	if err != nil {
		t.Fatalf("LoadRuntimePolicyFromPath() error = %v", err)
	}
	if got, want := label, string(runtimeArtifactKindGoMLXCritic); got != want {
		t.Fatalf("LoadRuntimePolicyFromPath() label = %q, want %q", got, want)
	}
	if _, ok := policy.(*GoMLXCriticRuntimePolicy); !ok {
		t.Fatalf("LoadRuntimePolicyFromPath() policy type = %T, want *GoMLXCriticRuntimePolicy", policy)
	}
}

func testGoMLXCriticManifest(checkpointDir string, hiddenDims []int) goMLXCriticRuntimeManifest {
	spec := DefaultTransitionNormalizationSpec()
	return goMLXCriticRuntimeManifest{
		Version:           goMLXCriticManifestVersion,
		TrainedAt:         time.Date(2026, time.March, 13, 0, 31, 28, 0, time.UTC),
		NormalizationSpec: spec,
		ObsDim:            spec.ObservationDim(),
		ActionDim:         spec.ActionDim(),
		InputDim:          spec.ObservationDim() + spec.ActionDim(),
		HiddenDims:        append([]int(nil), hiddenDims...),
		GlobalStep:        64080,
		CheckpointDir:     checkpointDir,
	}
}

func writeTestGoMLXCriticManifest(path string, manifest goMLXCriticRuntimeManifest) error {
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func writeZeroedTestGoMLXCheckpoint(basePath string, inputDim int, hiddenDims []int, outputBias float32) error {
	previousDim := inputDim
	tensors := make([]testGoMLXTensorPayload, 0, len(hiddenDims)*2+2)
	for index, hiddenDim := range hiddenDims {
		tensors = append(tensors,
			testGoMLXTensorPayload{
				Name:       "var:/hidden_" + strconv.Itoa(index) + "/dense/weights",
				Dimensions: []int{previousDim, hiddenDim},
				Values:     make([]float32, previousDim*hiddenDim),
			},
			testGoMLXTensorPayload{
				Name:       "var:/hidden_" + strconv.Itoa(index) + "/dense/biases",
				Dimensions: []int{hiddenDim},
				Values:     make([]float32, hiddenDim),
			},
		)
		previousDim = hiddenDim
	}
	tensors = append(tensors,
		testGoMLXTensorPayload{
			Name:       "var:/value_head/dense/weights",
			Dimensions: []int{previousDim, 1},
			Values:     make([]float32, previousDim),
		},
		testGoMLXTensorPayload{
			Name:       "var:/value_head/dense/biases",
			Dimensions: []int{1},
			Values:     []float32{outputBias},
		},
	)
	return writeTestGoMLXCheckpoint(basePath, tensors)
}

type testGoMLXTensorPayload struct {
	Name       string
	Dimensions []int
	Values     []float32
}

func writeTestGoMLXCheckpoint(basePath string, tensors []testGoMLXTensorPayload) error {
	var (
		rawData  bytes.Buffer
		metadata goMLXCheckpointMetadata
	)
	metadata.BinFormat = "gzip"
	for _, tensor := range tensors {
		elementCount := 1
		for _, dim := range tensor.Dimensions {
			elementCount *= dim
		}
		if len(tensor.Values) != elementCount {
			return os.ErrInvalid
		}

		pos := rawData.Len()
		for _, value := range tensor.Values {
			var encoded [4]byte
			binary.LittleEndian.PutUint32(encoded[:], math.Float32bits(value))
			if _, err := rawData.Write(encoded[:]); err != nil {
				return err
			}
		}
		metadata.Variables = append(metadata.Variables, goMLXCheckpointVariable{
			ParameterName: tensor.Name,
			Dimensions:    append([]int(nil), tensor.Dimensions...),
			DType:         "Float32",
			Pos:           pos,
			Length:        len(tensor.Values) * 4,
		})
	}

	metadataPayload, err := json.MarshalIndent(metadata, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(basePath+".json", metadataPayload, 0o644); err != nil {
		return err
	}

	var bin bytes.Buffer
	bin.WriteString(goMLXCheckpointBinHeader)
	bin.WriteByte(byte(len(goMLXCheckpointGZIPHeader)))
	bin.WriteString(goMLXCheckpointGZIPHeader)

	gzipWriter := gzip.NewWriter(&bin)
	if _, err := gzipWriter.Write(rawData.Bytes()); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	return os.WriteFile(basePath+".bin", bin.Bytes(), 0o644)
}
