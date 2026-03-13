package rl

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadTrainingTransitionsJSONLReadsRowsInOrder(t *testing.T) {
	recordOne := sampleTrainingTransitionRecord()
	recordOne.EpisodeID = 11
	recordOne.Tick = 3

	recordTwo := sampleTrainingTransitionRecord()
	recordTwo.EpisodeID = 11
	recordTwo.Tick = 4
	recordTwo.Done = 0

	payload := strings.Join([]string{
		mustMarshalTrainingTransitionJSONLRecord(t, recordOne),
		mustMarshalTrainingTransitionJSONLRecord(t, recordTwo),
		"",
	}, "\n")

	records, err := ReadTrainingTransitionsJSONL(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("ReadTrainingTransitionsJSONL() error = %v", err)
	}
	if got, want := len(records), 2; got != want {
		t.Fatalf("len(records) = %d, want %d", got, want)
	}
	if got, want := records[0].EpisodeID, uint64(11); got != want {
		t.Fatalf("records[0].EpisodeID = %d, want %d", got, want)
	}
	if got, want := records[1].Tick, uint32(4); got != want {
		t.Fatalf("records[1].Tick = %d, want %d", got, want)
	}
}

func mustMarshalTrainingTransitionJSONLRecord(t *testing.T, record TrainingTransitionRecord) string {
	t.Helper()
	summary := &strings.Builder{}
	encoder := json.NewEncoder(summary)
	if err := encoder.Encode(record); err != nil {
		t.Fatalf("Encode(record) error = %v", err)
	}
	return strings.TrimSpace(summary.String())
}
