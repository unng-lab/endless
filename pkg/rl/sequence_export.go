package rl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var errTransitionSequenceStreamStopped = errors.New("transition sequence stream stopped")

// TrainingTransitionSequenceQuery extends the stable row-oriented transition query with episode
// grouping controls for sequence trainers that consume whole episodes or fixed-size windows.
type TrainingTransitionSequenceQuery struct {
	TransitionQuery   TrainingTransitionQuery
	EpisodeLimit      int
	MaxSequenceLength int
}

// TrainingTransitionSequence keeps one grouped sequence export payload. When MaxSequenceLength
// is positive, one episode may be emitted as multiple windows distinguished by SequenceIndex.
type TrainingTransitionSequence struct {
	EpisodeID     uint64                     `json:"episode_id"`
	Scenario      string                     `json:"scenario"`
	Outcome       string                     `json:"outcome"`
	SequenceIndex int                        `json:"sequence_index"`
	StartTick     uint32                     `json:"start_tick"`
	EndTick       uint32                     `json:"end_tick"`
	StepCount     int                        `json:"step_count"`
	Transitions   []TrainingTransitionRecord `json:"transitions"`
}

// TransitionSequenceExportSummary reports the grouped export shape so CLI tooling can audit
// how many sequences, episodes and transition rows were emitted for sequence consumers.
type TransitionSequenceExportSummary struct {
	EpisodesExported  int
	SequencesExported int
	RowsExported      int
	Format            TransitionExportFormat
}

type transitionSequenceStreamBuilder struct {
	query             TrainingTransitionSequenceQuery
	visit             func(TrainingTransitionSequence) error
	current           TrainingTransitionSequence
	hasCurrent        bool
	episodesStarted   int
	sequencesExported int
	currentEpisodeID  uint64
}

// StreamTransitionSequences groups deterministic trainer-facing rows into episode-oriented
// sequences so sequence models can reuse the same filtered ClickHouse read path as row export.
func (r *ClickHouseTransitionReader) StreamTransitionSequences(ctx context.Context, query TrainingTransitionSequenceQuery, visit func(TrainingTransitionSequence) error) (TransitionSequenceExportSummary, error) {
	if r == nil || r.conn == nil {
		return TransitionSequenceExportSummary{}, fmt.Errorf("clickhouse transition reader is nil")
	}
	if visit == nil {
		return TransitionSequenceExportSummary{}, fmt.Errorf("transition sequence visitor is nil")
	}

	query = normalizedTrainingTransitionSequenceQuery(query)
	builder := newTransitionSequenceStreamBuilder(query, visit)
	transitionSummary, err := r.StreamTransitions(ctx, query.TransitionQuery, func(record TrainingTransitionRecord) error {
		return builder.Append(record)
	})
	if err != nil && !errors.Is(err, errTransitionSequenceStreamStopped) {
		return TransitionSequenceExportSummary{}, err
	}
	if err := builder.Flush(); err != nil && !errors.Is(err, errTransitionSequenceStreamStopped) {
		return TransitionSequenceExportSummary{}, err
	}

	return TransitionSequenceExportSummary{
		EpisodesExported:  builder.episodesStarted,
		SequencesExported: builder.sequencesExported,
		RowsExported:      transitionSummary.RowsExported,
	}, nil
}

// ExportTransitionSequences writes episode-grouped or fixed-window grouped transitions as jsonl
// or json so sequence trainers can consume a stable format without reconstructing episodes from
// raw SQL result ordering themselves.
func (r *ClickHouseTransitionReader) ExportTransitionSequences(ctx context.Context, writer io.Writer, query TrainingTransitionSequenceQuery, format TransitionExportFormat) (TransitionSequenceExportSummary, error) {
	if writer == nil {
		return TransitionSequenceExportSummary{}, fmt.Errorf("transition sequence export writer is nil")
	}

	format, err := normalizedTransitionExportFormat(format)
	if err != nil {
		return TransitionSequenceExportSummary{}, err
	}

	bufferedWriter := bufio.NewWriter(writer)
	encoder := json.NewEncoder(bufferedWriter)
	encoder.SetEscapeHTML(false)

	switch format {
	case TransitionExportFormatJSONL:
		summary, err := r.StreamTransitionSequences(ctx, query, func(sequence TrainingTransitionSequence) error {
			if err := encoder.Encode(sequence); err != nil {
				return fmt.Errorf("encode transition sequence jsonl row: %w", err)
			}
			return nil
		})
		summary.Format = format
		if err != nil {
			return summary, err
		}
		if err := bufferedWriter.Flush(); err != nil {
			return summary, fmt.Errorf("flush transition sequence jsonl export: %w", err)
		}
		return summary, nil
	case TransitionExportFormatJSON:
		if _, err := bufferedWriter.WriteString("[\n"); err != nil {
			return TransitionSequenceExportSummary{}, fmt.Errorf("write transition sequence json prefix: %w", err)
		}
		firstRecord := true
		summary, err := r.StreamTransitionSequences(ctx, query, func(sequence TrainingTransitionSequence) error {
			if !firstRecord {
				if _, err := bufferedWriter.WriteString(",\n"); err != nil {
					return fmt.Errorf("write transition sequence json separator: %w", err)
				}
			}
			payload, err := json.Marshal(sequence)
			if err != nil {
				return fmt.Errorf("marshal transition sequence json row: %w", err)
			}
			if _, err := bufferedWriter.Write(payload); err != nil {
				return fmt.Errorf("write transition sequence json row: %w", err)
			}
			firstRecord = false
			return nil
		})
		summary.Format = format
		if err != nil {
			return summary, err
		}
		if !firstRecord {
			if _, err := bufferedWriter.WriteString("\n"); err != nil {
				return summary, fmt.Errorf("write transition sequence json trailing newline: %w", err)
			}
		}
		if _, err := bufferedWriter.WriteString("]\n"); err != nil {
			return summary, fmt.Errorf("write transition sequence json suffix: %w", err)
		}
		if err := bufferedWriter.Flush(); err != nil {
			return summary, fmt.Errorf("flush transition sequence json export: %w", err)
		}
		return summary, nil
	default:
		return TransitionSequenceExportSummary{}, fmt.Errorf("unsupported transition sequence export format %q", format)
	}
}

func normalizedTrainingTransitionSequenceQuery(query TrainingTransitionSequenceQuery) TrainingTransitionSequenceQuery {
	query.TransitionQuery = normalizedTrainingTransitionQuery(query.TransitionQuery)
	if query.EpisodeLimit < 0 {
		query.EpisodeLimit = 0
	}
	if query.MaxSequenceLength < 0 {
		query.MaxSequenceLength = 0
	}
	return query
}

func newTransitionSequenceStreamBuilder(query TrainingTransitionSequenceQuery, visit func(TrainingTransitionSequence) error) *transitionSequenceStreamBuilder {
	return &transitionSequenceStreamBuilder{
		query: query,
		visit: visit,
	}
}

func (b *transitionSequenceStreamBuilder) Append(record TrainingTransitionRecord) error {
	if b == nil {
		return fmt.Errorf("transition sequence stream builder is nil")
	}

	if !b.hasCurrent {
		return b.startNewEpisodeSequence(record)
	}

	if record.EpisodeID != b.currentEpisodeID {
		if err := b.flushCurrent(); err != nil {
			return err
		}
		return b.startNewEpisodeSequence(record)
	}

	if b.query.MaxSequenceLength > 0 && len(b.current.Transitions) >= b.query.MaxSequenceLength {
		nextSequenceIndex := b.current.SequenceIndex + 1
		if err := b.flushCurrent(); err != nil {
			return err
		}
		return b.startNextWindow(record, nextSequenceIndex)
	}

	b.appendRecord(record)
	return nil
}

func (b *transitionSequenceStreamBuilder) Flush() error {
	if b == nil || !b.hasCurrent {
		return nil
	}
	return b.flushCurrent()
}

func (b *transitionSequenceStreamBuilder) startNewEpisodeSequence(record TrainingTransitionRecord) error {
	if b.query.EpisodeLimit > 0 && b.episodesStarted >= b.query.EpisodeLimit {
		return errTransitionSequenceStreamStopped
	}
	b.episodesStarted++
	b.currentEpisodeID = record.EpisodeID
	b.current = TrainingTransitionSequence{
		EpisodeID:     record.EpisodeID,
		Scenario:      record.Scenario,
		Outcome:       record.Outcome,
		SequenceIndex: 0,
		Transitions:   make([]TrainingTransitionRecord, 0, maxSequenceCapacity(1, b.query.MaxSequenceLength)),
	}
	b.hasCurrent = true
	b.appendRecord(record)
	return nil
}

func (b *transitionSequenceStreamBuilder) startNextWindow(record TrainingTransitionRecord, sequenceIndex int) error {
	b.current = TrainingTransitionSequence{
		EpisodeID:     record.EpisodeID,
		Scenario:      record.Scenario,
		Outcome:       record.Outcome,
		SequenceIndex: sequenceIndex,
		Transitions:   make([]TrainingTransitionRecord, 0, maxSequenceCapacity(1, b.query.MaxSequenceLength)),
	}
	b.hasCurrent = true
	b.currentEpisodeID = record.EpisodeID
	b.appendRecord(record)
	return nil
}

func (b *transitionSequenceStreamBuilder) appendRecord(record TrainingTransitionRecord) {
	if len(b.current.Transitions) == 0 {
		b.current.StartTick = record.Tick
	}
	b.current.EndTick = record.Tick
	b.current.StepCount++
	b.current.Transitions = append(b.current.Transitions, record)
}

func (b *transitionSequenceStreamBuilder) flushCurrent() error {
	if !b.hasCurrent {
		return nil
	}
	if b.visit == nil {
		return fmt.Errorf("transition sequence visitor is nil")
	}
	sequence := b.current
	b.hasCurrent = false
	if err := b.visit(sequence); err != nil {
		return err
	}
	b.sequencesExported++
	return nil
}

func maxSequenceCapacity(left, right int) int {
	if left > right {
		return left
	}
	return right
}
