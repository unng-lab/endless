package gomlxtrain

import (
	"time"

	"github.com/unng-lab/endless/pkg/rl"
)

// InputFormat identifies which stable trainer-facing source should be used to feed the
// external GoMLX trainer.
type InputFormat string

const (
	InputFormatJSONL      InputFormat = "jsonl"
	InputFormatClickHouse InputFormat = "clickhouse"
)

// InputSourceConfig describes where the trainer should read transition rows from. The same
// logical query filters apply both to ClickHouse reads and to already exported JSONL slices.
type InputSourceConfig struct {
	Format InputFormat                `json:"format"`
	Path   string                     `json:"path,omitempty"`
	Query  rl.TrainingTransitionQuery `json:"query"`
}

// Config defines all operator-facing knobs for the external GoMLX critic trainer.
type Config struct {
	Source         InputSourceConfig
	CheckpointDir  string
	CheckpointKeep int
	Backend        string
	BatchSize      int
	Epochs         int
	LearningRate   float64
	Discount       float32
	TargetAbsMax   float32
	HiddenDims     []int
	Seed           int64
}

// PreparedDataset contains the fully materialized critic-training tensors and the summary needed
// by both the GoMLX trainer and the manifest writer.
type PreparedDataset struct {
	Inputs                  []float32
	Targets                 []float32
	NormalizationSpec       rl.TransitionNormalizationSpec
	ObservationFeatureNames []string
	ActionFeatureNames      []string
	Samples                 int
	ObsDim                  int
	ActionDim               int
	InputDim                int
	TargetMin               float32
	TargetMax               float32
	ContinuedTransitions    int
	TerminalTransitions     int
	UnlinkedTransitions     int
}

// Manifest persists the exact tensor contract, trainer source and GoMLX hyperparameters used to
// produce one checkpoint directory.
type Manifest struct {
	Version                 int                            `json:"version"`
	TrainedAt               time.Time                      `json:"trained_at"`
	Source                  InputSourceConfig              `json:"source"`
	NormalizationSpec       rl.TransitionNormalizationSpec `json:"normalization_spec"`
	ObservationFeatureNames []string                       `json:"observation_feature_names"`
	ActionFeatureNames      []string                       `json:"action_feature_names"`
	ObsDim                  int                            `json:"obs_dim"`
	ActionDim               int                            `json:"action_dim"`
	InputDim                int                            `json:"input_dim"`
	HiddenDims              []int                          `json:"hidden_dims"`
	Samples                 int                            `json:"samples"`
	ContinuedTransitions    int                            `json:"continued_transitions"`
	TerminalTransitions     int                            `json:"terminal_transitions"`
	UnlinkedTransitions     int                            `json:"unlinked_transitions"`
	TargetMin               float32                        `json:"target_min"`
	TargetMax               float32                        `json:"target_max"`
	BatchSize               int                            `json:"batch_size"`
	Epochs                  int                            `json:"epochs"`
	LearningRate            float64                        `json:"learning_rate"`
	Discount                float32                        `json:"discount"`
	TargetAbsMax            float32                        `json:"target_abs_max"`
	Seed                    int64                          `json:"seed"`
	Backend                 string                         `json:"backend"`
	BackendName             string                         `json:"backend_name"`
	BackendDescription      string                         `json:"backend_description"`
	GlobalStep              int64                          `json:"global_step"`
	CheckpointDir           string                         `json:"checkpoint_dir"`
}

// Result reports the outcome of one trainer invocation so the CLI may log stable audit lines.
type Result struct {
	BackendName          string
	BackendDescription   string
	ManifestPath         string
	CheckpointDir        string
	Samples              int
	ObsDim               int
	ActionDim            int
	InputDim             int
	ContinuedTransitions int
	TerminalTransitions  int
	UnlinkedTransitions  int
	TargetMin            float32
	TargetMax            float32
	StartStep            int64
	EndStep              int64
}
