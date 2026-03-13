package rl

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	goMLXCriticManifestFileName = "gomlx_critic_manifest.json"
	goMLXCriticManifestVersion  = 1
	goMLXCheckpointBinHeader    = "gomlx_checkpoints"
	goMLXCheckpointGZIPHeader   = "gzip"
)

// GoMLXCriticRuntimePolicy evaluates the trainer-side MLP critic directly in pure Go so the
// desktop duel visualizer can replay GoMLX checkpoints without linking the full training stack.
type GoMLXCriticRuntimePolicy struct {
	artifact goMLXCriticRuntimeArtifact
	fallback Policy

	lastDecisionDebug string
}

type goMLXCriticRuntimeArtifact struct {
	Manifest           goMLXCriticRuntimeManifest
	Model              goMLXCriticRuntimeModel
	ManifestPath       string
	CheckpointJSONPath string
	CheckpointBINPath  string
}

type goMLXCriticRuntimeManifest struct {
	Version           int                         `json:"version"`
	TrainedAt         time.Time                   `json:"trained_at"`
	NormalizationSpec TransitionNormalizationSpec `json:"normalization_spec"`
	ObsDim            int                         `json:"obs_dim"`
	ActionDim         int                         `json:"action_dim"`
	InputDim          int                         `json:"input_dim"`
	HiddenDims        []int                       `json:"hidden_dims"`
	GlobalStep        int64                       `json:"global_step"`
	CheckpointDir     string                      `json:"checkpoint_dir"`
}

type goMLXCheckpointMetadata struct {
	Variables []goMLXCheckpointVariable `json:"Variables"`
	BinFormat string                    `json:"BinFormat"`
}

type goMLXCheckpointVariable struct {
	ParameterName string `json:"ParameterName"`
	Dimensions    []int  `json:"Dimensions"`
	DType         string `json:"DType"`
	Pos           int    `json:"Pos"`
	Length        int    `json:"Length"`
}

type goMLXCriticDenseLayer struct {
	Weights []float32
	Biases  []float32
	InDim   int
	OutDim  int
}

type goMLXCriticRuntimeModel struct {
	InputDim     int
	HiddenDims   []int
	HiddenLayers []goMLXCriticDenseLayer
	OutputLayer  goMLXCriticDenseLayer
}

type resolvedGoMLXCriticArtifactPaths struct {
	ManifestPath       string
	CheckpointJSONPath string
	CheckpointBINPath  string
}

// LoadGoMLXCriticRuntimePolicy restores one trainer manifest plus its newest checkpoint so the
// runtime can score candidate actions with the trained MLP critic.
func LoadGoMLXCriticRuntimePolicy(path string) (*GoMLXCriticRuntimePolicy, error) {
	artifact, err := loadGoMLXCriticRuntimeArtifact(path)
	if err != nil {
		return nil, err
	}
	return NewGoMLXCriticRuntimePolicy(artifact)
}

// NewGoMLXCriticRuntimePolicy validates the parsed checkpoint-derived artifact and attaches the
// same scripted fallback used by the simpler stub runtime scorer.
func NewGoMLXCriticRuntimePolicy(artifact goMLXCriticRuntimeArtifact) (*GoMLXCriticRuntimePolicy, error) {
	if err := artifact.Validate(); err != nil {
		return nil, err
	}

	return &GoMLXCriticRuntimePolicy{
		artifact: artifact,
		fallback: NewLeadAndStrafePolicy(),
	}, nil
}

// Validate checks that the manifest, tensor contract and dense layer shapes still form one
// coherent runtime model before any gameplay code relies on them for inference.
func (a goMLXCriticRuntimeArtifact) Validate() error {
	if err := a.Manifest.Validate(); err != nil {
		return err
	}
	if err := a.Model.Validate(a.Manifest); err != nil {
		return err
	}
	return nil
}

// Validate confirms that the trainer manifest still matches the current runtime tensorization
// contract and that the saved hidden-layer declaration is internally consistent.
func (m goMLXCriticRuntimeManifest) Validate() error {
	if m.Version != goMLXCriticManifestVersion {
		return fmt.Errorf("GoMLX critic manifest version = %d, want %d", m.Version, goMLXCriticManifestVersion)
	}

	spec := m.NormalizationSpec.Normalized()
	if m.ObsDim != spec.ObservationDim() {
		return fmt.Errorf("GoMLX critic manifest observation dim = %d, want %d", m.ObsDim, spec.ObservationDim())
	}
	if m.ActionDim != spec.ActionDim() {
		return fmt.Errorf("GoMLX critic manifest action dim = %d, want %d", m.ActionDim, spec.ActionDim())
	}
	if m.InputDim != m.ObsDim+m.ActionDim {
		return fmt.Errorf("GoMLX critic manifest input dim = %d, want %d", m.InputDim, m.ObsDim+m.ActionDim)
	}
	for index, hiddenDim := range m.HiddenDims {
		if hiddenDim <= 0 {
			return fmt.Errorf("GoMLX critic manifest hidden_dims[%d] = %d, want positive width", index, hiddenDim)
		}
	}
	return nil
}

// Validate verifies every dense layer shape against the manifest-declared network layout so the
// pure-Go forward pass never silently runs with malformed checkpoint weights.
func (m goMLXCriticRuntimeModel) Validate(manifest goMLXCriticRuntimeManifest) error {
	if m.InputDim != manifest.InputDim {
		return fmt.Errorf("GoMLX critic model input dim = %d, want %d", m.InputDim, manifest.InputDim)
	}
	if len(m.HiddenLayers) != len(manifest.HiddenDims) {
		return fmt.Errorf("GoMLX critic model hidden layer count = %d, want %d", len(m.HiddenLayers), len(manifest.HiddenDims))
	}

	previousDim := manifest.InputDim
	for index, hiddenLayer := range m.HiddenLayers {
		expectedOutDim := manifest.HiddenDims[index]
		if err := hiddenLayer.validate(previousDim, expectedOutDim, fmt.Sprintf("hidden_%d", index)); err != nil {
			return err
		}
		previousDim = expectedOutDim
	}
	if err := m.OutputLayer.validate(previousDim, 1, "value_head"); err != nil {
		return err
	}
	return nil
}

func (l goMLXCriticDenseLayer) validate(expectedInDim, expectedOutDim int, label string) error {
	if l.InDim != expectedInDim {
		return fmt.Errorf("GoMLX critic %s input dim = %d, want %d", label, l.InDim, expectedInDim)
	}
	if l.OutDim != expectedOutDim {
		return fmt.Errorf("GoMLX critic %s output dim = %d, want %d", label, l.OutDim, expectedOutDim)
	}
	if len(l.Weights) != expectedInDim*expectedOutDim {
		return fmt.Errorf("GoMLX critic %s weights = %d, want %d", label, len(l.Weights), expectedInDim*expectedOutDim)
	}
	if len(l.Biases) != expectedOutDim {
		return fmt.Errorf("GoMLX critic %s biases = %d, want %d", label, len(l.Biases), expectedOutDim)
	}
	return nil
}

func loadGoMLXCriticRuntimeArtifact(path string) (goMLXCriticRuntimeArtifact, error) {
	resolved, err := resolveGoMLXCriticArtifactPaths(path)
	if err != nil {
		return goMLXCriticRuntimeArtifact{}, err
	}

	manifest, err := loadGoMLXCriticRuntimeManifest(resolved.ManifestPath)
	if err != nil {
		return goMLXCriticRuntimeArtifact{}, err
	}

	metadata, checkpointData, err := loadGoMLXCheckpointBundle(resolved.CheckpointJSONPath, resolved.CheckpointBINPath)
	if err != nil {
		return goMLXCriticRuntimeArtifact{}, err
	}

	model, err := buildGoMLXCriticRuntimeModel(manifest, metadata, checkpointData)
	if err != nil {
		return goMLXCriticRuntimeArtifact{}, err
	}

	artifact := goMLXCriticRuntimeArtifact{
		Manifest:           manifest,
		Model:              model,
		ManifestPath:       resolved.ManifestPath,
		CheckpointJSONPath: resolved.CheckpointJSONPath,
		CheckpointBINPath:  resolved.CheckpointBINPath,
	}
	if err := artifact.Validate(); err != nil {
		return goMLXCriticRuntimeArtifact{}, err
	}
	return artifact, nil
}

func resolveGoMLXCriticArtifactPaths(path string) (resolvedGoMLXCriticArtifactPaths, error) {
	info, err := os.Stat(path)
	if err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("stat GoMLX critic artifact path %q: %w", path, err)
	}
	if info.IsDir() {
		return resolveGoMLXCriticArtifactDirectory(path)
	}

	lowerPath := strings.ToLower(path)
	switch filepath.Ext(lowerPath) {
	case ".bin":
		return resolveGoMLXCriticCheckpointBinary(path)
	case ".json":
		root, err := loadRuntimeArtifactJSONRoot(path)
		if err != nil {
			return resolvedGoMLXCriticArtifactPaths{}, err
		}
		switch {
		case hasJSONKeys(root, "trained_at", "obs_dim", "action_dim", "input_dim", "checkpoint_dir"):
			return resolveGoMLXCriticManifest(path)
		case hasJSONKeys(root, "Variables", "BinFormat"):
			return resolveGoMLXCriticCheckpointMetadata(path)
		default:
			return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("JSON file %q is neither a GoMLX critic manifest nor GoMLX checkpoint metadata", path)
		}
	default:
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("GoMLX critic artifact %q must be a checkpoint directory, manifest JSON, checkpoint metadata JSON, or checkpoint BIN file", path)
	}
}

func resolveGoMLXCriticArtifactDirectory(dir string) (resolvedGoMLXCriticArtifactPaths, error) {
	manifestPath := filepath.Join(dir, goMLXCriticManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("GoMLX critic artifact directory %q does not contain %q: %w", dir, goMLXCriticManifestFileName, err)
	}
	checkpointBase, err := latestGoMLXCheckpointBase(dir)
	if err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, err
	}
	return resolvedGoMLXCriticArtifactPaths{
		ManifestPath:       manifestPath,
		CheckpointJSONPath: filepath.Join(dir, checkpointBase+".json"),
		CheckpointBINPath:  filepath.Join(dir, checkpointBase+".bin"),
	}, nil
}

func resolveGoMLXCriticManifest(path string) (resolvedGoMLXCriticArtifactPaths, error) {
	dir := filepath.Dir(path)
	checkpointBase, err := latestGoMLXCheckpointBase(dir)
	if err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, err
	}
	return resolvedGoMLXCriticArtifactPaths{
		ManifestPath:       path,
		CheckpointJSONPath: filepath.Join(dir, checkpointBase+".json"),
		CheckpointBINPath:  filepath.Join(dir, checkpointBase+".bin"),
	}, nil
}

func resolveGoMLXCriticCheckpointMetadata(path string) (resolvedGoMLXCriticArtifactPaths, error) {
	dir := filepath.Dir(path)
	manifestPath := filepath.Join(dir, goMLXCriticManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("GoMLX checkpoint metadata %q requires sibling manifest %q: %w", path, manifestPath, err)
	}
	binPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".bin"
	if _, err := os.Stat(binPath); err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("GoMLX checkpoint metadata %q requires binary payload %q: %w", path, binPath, err)
	}
	return resolvedGoMLXCriticArtifactPaths{
		ManifestPath:       manifestPath,
		CheckpointJSONPath: path,
		CheckpointBINPath:  binPath,
	}, nil
}

func resolveGoMLXCriticCheckpointBinary(path string) (resolvedGoMLXCriticArtifactPaths, error) {
	dir := filepath.Dir(path)
	manifestPath := filepath.Join(dir, goMLXCriticManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("GoMLX checkpoint binary %q requires sibling manifest %q: %w", path, manifestPath, err)
	}
	jsonPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".json"
	if _, err := os.Stat(jsonPath); err != nil {
		return resolvedGoMLXCriticArtifactPaths{}, fmt.Errorf("GoMLX checkpoint binary %q requires metadata %q: %w", path, jsonPath, err)
	}
	return resolvedGoMLXCriticArtifactPaths{
		ManifestPath:       manifestPath,
		CheckpointJSONPath: jsonPath,
		CheckpointBINPath:  path,
	}, nil
}

func latestGoMLXCheckpointBase(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read GoMLX checkpoint directory %q: %w", dir, err)
	}

	bases := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "checkpoint-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		base := strings.TrimSuffix(name, ".json")
		if _, err := os.Stat(filepath.Join(dir, base+".bin")); err != nil {
			continue
		}
		bases = append(bases, base)
	}
	if len(bases) == 0 {
		return "", fmt.Errorf("GoMLX checkpoint directory %q does not contain any checkpoint-*.json/.bin pair", dir)
	}
	sort.Strings(bases)
	return bases[len(bases)-1], nil
}

func loadGoMLXCriticRuntimeManifest(path string) (goMLXCriticRuntimeManifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return goMLXCriticRuntimeManifest{}, fmt.Errorf("read GoMLX critic manifest %q: %w", path, err)
	}

	var manifest goMLXCriticRuntimeManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return goMLXCriticRuntimeManifest{}, fmt.Errorf("decode GoMLX critic manifest %q: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return goMLXCriticRuntimeManifest{}, err
	}
	return manifest, nil
}

func loadGoMLXCheckpointBundle(jsonPath, binPath string) (goMLXCheckpointMetadata, []byte, error) {
	metadataPayload, err := os.ReadFile(jsonPath)
	if err != nil {
		return goMLXCheckpointMetadata{}, nil, fmt.Errorf("read GoMLX checkpoint metadata %q: %w", jsonPath, err)
	}

	var metadata goMLXCheckpointMetadata
	if err := json.Unmarshal(metadataPayload, &metadata); err != nil {
		return goMLXCheckpointMetadata{}, nil, fmt.Errorf("decode GoMLX checkpoint metadata %q: %w", jsonPath, err)
	}

	checkpointData, err := readGoMLXCheckpointBinary(binPath)
	if err != nil {
		return goMLXCheckpointMetadata{}, nil, err
	}
	return metadata, checkpointData, nil
}

// readGoMLXCheckpointBinary understands the lightweight header used by GoMLX checkpoints and
// returns the fully decompressed tensor byte stream expected by the JSON metadata offsets.
func readGoMLXCheckpointBinary(path string) ([]byte, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read GoMLX checkpoint binary %q: %w", path, err)
	}
	reader := bytes.NewReader(payload)

	header := make([]byte, len(goMLXCheckpointBinHeader))
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, fmt.Errorf("read GoMLX checkpoint binary header %q: %w", path, err)
	}
	if string(header) != goMLXCheckpointBinHeader {
		return payload, nil
	}

	var compressionHeaderLength uint8
	if err := binary.Read(reader, binary.BigEndian, &compressionHeaderLength); err != nil {
		return nil, fmt.Errorf("read GoMLX checkpoint compression header %q: %w", path, err)
	}
	compressionHeader := make([]byte, compressionHeaderLength)
	if _, err := io.ReadFull(reader, compressionHeader); err != nil {
		return nil, fmt.Errorf("read GoMLX checkpoint compression name %q: %w", path, err)
	}
	if string(compressionHeader) != goMLXCheckpointGZIPHeader {
		return nil, fmt.Errorf("GoMLX checkpoint binary %q uses unsupported compression %q", path, string(compressionHeader))
	}

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open GoMLX checkpoint gzip payload %q: %w", path, err)
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("decompress GoMLX checkpoint binary %q: %w", path, err)
	}
	return decompressed, nil
}

func buildGoMLXCriticRuntimeModel(manifest goMLXCriticRuntimeManifest, metadata goMLXCheckpointMetadata, checkpointData []byte) (goMLXCriticRuntimeModel, error) {
	model := goMLXCriticRuntimeModel{
		InputDim:   manifest.InputDim,
		HiddenDims: append([]int(nil), manifest.HiddenDims...),
	}

	previousDim := manifest.InputDim
	for index, hiddenDim := range manifest.HiddenDims {
		layer, err := loadGoMLXCriticDenseLayer(metadata, checkpointData, fmt.Sprintf("hidden_%d", index), previousDim, hiddenDim)
		if err != nil {
			return goMLXCriticRuntimeModel{}, err
		}
		model.HiddenLayers = append(model.HiddenLayers, layer)
		previousDim = hiddenDim
	}

	outputLayer, err := loadGoMLXCriticDenseLayer(metadata, checkpointData, "value_head", previousDim, 1)
	if err != nil {
		return goMLXCriticRuntimeModel{}, err
	}
	model.OutputLayer = outputLayer
	return model, nil
}

func loadGoMLXCriticDenseLayer(metadata goMLXCheckpointMetadata, checkpointData []byte, scope string, inputDim, outputDim int) (goMLXCriticDenseLayer, error) {
	weightsName := fmt.Sprintf("var:/%s/dense/weights", scope)
	biasesName := fmt.Sprintf("var:/%s/dense/biases", scope)

	weights, err := loadGoMLXCheckpointFloat32(metadata, checkpointData, weightsName, inputDim, outputDim)
	if err != nil {
		return goMLXCriticDenseLayer{}, err
	}
	biases, err := loadGoMLXCheckpointFloat32(metadata, checkpointData, biasesName, outputDim)
	if err != nil {
		return goMLXCriticDenseLayer{}, err
	}

	layer := goMLXCriticDenseLayer{
		Weights: weights,
		Biases:  biases,
		InDim:   inputDim,
		OutDim:  outputDim,
	}
	if err := layer.validate(inputDim, outputDim, scope); err != nil {
		return goMLXCriticDenseLayer{}, err
	}
	return layer, nil
}

func loadGoMLXCheckpointFloat32(metadata goMLXCheckpointMetadata, checkpointData []byte, parameterName string, expectedDims ...int) ([]float32, error) {
	var variable *goMLXCheckpointVariable
	for index := range metadata.Variables {
		if metadata.Variables[index].ParameterName == parameterName {
			variable = &metadata.Variables[index]
			break
		}
	}
	if variable == nil {
		return nil, fmt.Errorf("GoMLX checkpoint is missing variable %q", parameterName)
	}
	if strings.ToLower(variable.DType) != "float32" {
		return nil, fmt.Errorf("GoMLX checkpoint variable %q dtype = %s, want Float32", parameterName, variable.DType)
	}
	if len(variable.Dimensions) != len(expectedDims) {
		return nil, fmt.Errorf("GoMLX checkpoint variable %q rank = %d, want %d", parameterName, len(variable.Dimensions), len(expectedDims))
	}

	elementCount := 1
	for index, dim := range expectedDims {
		if variable.Dimensions[index] != dim {
			return nil, fmt.Errorf("GoMLX checkpoint variable %q dim[%d] = %d, want %d", parameterName, index, variable.Dimensions[index], dim)
		}
		elementCount *= dim
	}

	expectedLength := elementCount * 4
	if variable.Length != expectedLength {
		return nil, fmt.Errorf("GoMLX checkpoint variable %q length = %d, want %d", parameterName, variable.Length, expectedLength)
	}
	if variable.Pos < 0 || variable.Pos+variable.Length > len(checkpointData) {
		return nil, fmt.Errorf("GoMLX checkpoint variable %q range [%d,%d) is outside binary payload length %d", parameterName, variable.Pos, variable.Pos+variable.Length, len(checkpointData))
	}

	values := make([]float32, elementCount)
	data := checkpointData[variable.Pos : variable.Pos+variable.Length]
	for index := range values {
		raw := binary.LittleEndian.Uint32(data[index*4:])
		values[index] = math.Float32frombits(raw)
	}
	return values, nil
}

// Predict executes the exact dense+ReLU+dense stack described by the trainer manifest using
// the weights extracted from the selected GoMLX checkpoint.
func (m goMLXCriticRuntimeModel) Predict(input []float32) (float32, error) {
	if len(input) != m.InputDim {
		return 0, fmt.Errorf("GoMLX critic input dim = %d, want %d", len(input), m.InputDim)
	}

	current := append([]float32(nil), input...)
	for _, hiddenLayer := range m.HiddenLayers {
		next := make([]float32, hiddenLayer.OutDim)
		for outputIndex := 0; outputIndex < hiddenLayer.OutDim; outputIndex++ {
			value := hiddenLayer.Biases[outputIndex]
			for inputIndex := 0; inputIndex < hiddenLayer.InDim; inputIndex++ {
				value += current[inputIndex] * hiddenLayer.Weights[inputIndex*hiddenLayer.OutDim+outputIndex]
			}
			if value < 0 {
				value = 0
			}
			next[outputIndex] = value
		}
		current = next
	}

	value := m.OutputLayer.Biases[0]
	for inputIndex := 0; inputIndex < m.OutputLayer.InDim; inputIndex++ {
		value += current[inputIndex] * m.OutputLayer.Weights[inputIndex*m.OutputLayer.OutDim]
	}
	if !isFiniteFloat32(value) {
		return 0, fmt.Errorf("GoMLX critic prediction became non-finite")
	}
	return value, nil
}

// ChooseAction scores the same conservative candidate set used by the linear stub runtime path
// but replaces the value function with the MLP critic exported from the GoMLX checkpoint.
func (p *GoMLXCriticRuntimePolicy) ChooseAction(observation Observation) Action {
	if p == nil {
		return Action{Type: ActionTypeNone}
	}

	spec := p.artifact.Manifest.NormalizationSpec.Normalized()
	obsVector, err := vectorizeRuntimeObservation(observation, spec)
	if err != nil {
		p.lastDecisionDebug = "model: observation vectorization failed, fallback policy used"
		return p.fallbackAction(observation)
	}

	candidates := buildLinearQStubActionCandidates(observation)
	if len(candidates) == 0 {
		p.lastDecisionDebug = "model: no candidates, fallback policy used"
		return p.fallbackAction(observation)
	}

	bestAction := candidates[0]
	bestScore := float32(math.Inf(-1))
	successfulCandidates := 0
	for _, candidate := range candidates {
		actionVector, err := vectorizeRuntimeAction(spec, candidate, true)
		if err != nil {
			continue
		}

		input := make([]float32, 0, len(obsVector)+len(actionVector))
		input = append(input, obsVector...)
		input = append(input, actionVector...)
		score, err := p.artifact.Model.Predict(input)
		if err != nil {
			continue
		}
		successfulCandidates++
		if score > bestScore {
			bestScore = score
			bestAction = candidate
		}
	}

	if math.IsInf(float64(bestScore), -1) {
		p.lastDecisionDebug = "model: candidate scoring failed, fallback policy used"
		return p.fallbackAction(observation)
	}
	p.lastDecisionDebug = fmt.Sprintf(
		"model: gomlx best_score=%.4f candidates=%d scored=%d chosen=%s",
		bestScore,
		len(candidates),
		successfulCandidates,
		actionKey(bestAction),
	)
	return bestAction
}

// LastDecisionDebugText exposes one compact trace of the most recent GoMLX runtime scoring
// pass so the overlay can explain which candidate won and whether fallback was needed.
func (p *GoMLXCriticRuntimePolicy) LastDecisionDebugText() string {
	if p == nil {
		return ""
	}
	return p.lastDecisionDebug
}

func (p *GoMLXCriticRuntimePolicy) fallbackAction(observation Observation) Action {
	if p == nil || p.fallback == nil {
		return Action{Type: ActionTypeNone}
	}
	return p.fallback.ChooseAction(observation)
}
