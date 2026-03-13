package rl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const maxTrainingTransitionJSONLLineBytes = 1 << 20

// StreamTrainingTransitionsJSONL reads one trainer-facing transition per line from JSONL and
// forwards each decoded record to visit. This keeps the external trainer path symmetric with the
// ClickHouse streaming reader while still allowing file-based datasets to be processed linearly.
func StreamTrainingTransitionsJSONL(reader io.Reader, visit func(TrainingTransitionRecord) error) (TransitionExportSummary, error) {
	if reader == nil {
		return TransitionExportSummary{}, fmt.Errorf("training transition jsonl reader is nil")
	}
	if visit == nil {
		return TransitionExportSummary{}, fmt.Errorf("training transition jsonl visitor is nil")
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTrainingTransitionJSONLLineBytes)

	summary := TransitionExportSummary{Format: TransitionExportFormatJSONL}
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record TrainingTransitionRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return summary, fmt.Errorf("decode training transition jsonl row %d: %w", summary.RowsExported+1, err)
		}
		if err := visit(record); err != nil {
			return summary, err
		}
		summary.RowsExported++
	}
	if err := scanner.Err(); err != nil {
		return summary, fmt.Errorf("scan training transition jsonl: %w", err)
	}
	return summary, nil
}

// ReadTrainingTransitionsJSONL eagerly loads one JSONL file into memory. The external GoMLX
// trainer uses this to construct deterministic in-memory epochs over an exported dataset slice.
func ReadTrainingTransitionsJSONL(reader io.Reader) ([]TrainingTransitionRecord, error) {
	records := make([]TrainingTransitionRecord, 0)
	_, err := StreamTrainingTransitionsJSONL(reader, func(record TrainingTransitionRecord) error {
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}
