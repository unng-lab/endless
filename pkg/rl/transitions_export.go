package rl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// TransitionExportFormat fixes the on-disk shape emitted for the first external dataset
// consumer so trainer-side tooling may rely on one small set of stable file layouts.
type TransitionExportFormat string

const (
	TransitionExportFormatJSONL TransitionExportFormat = "jsonl"
	TransitionExportFormatJSON  TransitionExportFormat = "json"
)

// TrainingTransitionQuery collects the common filters needed by the first exporter without
// forcing external tools to assemble SQL manually around the trainer-facing transitions view.
type TrainingTransitionQuery struct {
	Scenario     string
	Outcome      string
	EpisodeIDMin uint64
	EpisodeIDMax uint64
	Limit        int
}

// TransitionExportSummary reports how many rows the exporter emitted so CLI tooling may print
// stable operator-facing progress and audit information after long exports finish.
type TransitionExportSummary struct {
	RowsExported int
	Format       TransitionExportFormat
}

// ClickHouseTransitionReader streams rows from the stable trainer-facing transitions view so
// exporters and future external readers share one deterministic read path.
type ClickHouseTransitionReader struct {
	conn driver.Conn
	cfg  ClickHouseConfig
}

var trainingTransitionSelectColumns = mustTrainingTransitionSelectColumns()

// NewClickHouseTransitionReader opens a dedicated ClickHouse connection for trainer-facing
// transition reads without coupling external consumers to the write-side recorder.
func NewClickHouseTransitionReader(ctx context.Context, cfg ClickHouseConfig) (*ClickHouseTransitionReader, error) {
	conn, cfg, err := openClickHouseConn(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &ClickHouseTransitionReader{
		conn: conn,
		cfg:  cfg,
	}, nil
}

// Close releases the reader connection once the caller no longer needs transition rows.
func (r *ClickHouseTransitionReader) Close() error {
	if r == nil || r.conn == nil {
		return nil
	}
	return r.conn.Close()
}

// StreamTransitions reads trainer-facing transition rows in deterministic episode/tick order
// and calls visit for each row so callers may export large datasets without holding them all
// in memory at once.
func (r *ClickHouseTransitionReader) StreamTransitions(ctx context.Context, query TrainingTransitionQuery, visit func(TrainingTransitionRecord) error) (TransitionExportSummary, error) {
	if r == nil || r.conn == nil {
		return TransitionExportSummary{}, fmt.Errorf("clickhouse transition reader is nil")
	}
	if visit == nil {
		return TransitionExportSummary{}, fmt.Errorf("transition visitor is nil")
	}

	query = normalizedTrainingTransitionQuery(query)
	selectQuery, args := buildTrainingTransitionsSelectQuery(r.cfg, query)
	rows, err := r.conn.Query(ctx, selectQuery, args...)
	if err != nil {
		return TransitionExportSummary{}, fmt.Errorf("query training transitions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	summary := TransitionExportSummary{}
	for rows.Next() {
		var record TrainingTransitionRecord
		if err := rows.ScanStruct(&record); err != nil {
			return summary, fmt.Errorf("scan training transition: %w", err)
		}
		if err := visit(record); err != nil {
			return summary, err
		}
		summary.RowsExported++
	}
	if err := rows.Err(); err != nil {
		return summary, fmt.Errorf("iterate training transitions: %w", err)
	}
	return summary, nil
}

// ExportTransitions writes the selected transition rows into a minimal trainer-friendly file
// format so the next external trainer can start from exported records instead of raw SQL.
func (r *ClickHouseTransitionReader) ExportTransitions(ctx context.Context, writer io.Writer, query TrainingTransitionQuery, format TransitionExportFormat) (TransitionExportSummary, error) {
	if writer == nil {
		return TransitionExportSummary{}, fmt.Errorf("transition export writer is nil")
	}

	format, err := normalizedTransitionExportFormat(format)
	if err != nil {
		return TransitionExportSummary{}, err
	}

	bufferedWriter := bufio.NewWriter(writer)

	encoder := json.NewEncoder(bufferedWriter)
	encoder.SetEscapeHTML(false)

	switch format {
	case TransitionExportFormatJSONL:
		summary, err := r.StreamTransitions(ctx, query, func(record TrainingTransitionRecord) error {
			if err := encoder.Encode(record); err != nil {
				return fmt.Errorf("encode transition jsonl row: %w", err)
			}
			return nil
		})
		summary.Format = format
		if err != nil {
			return summary, err
		}
		if err := bufferedWriter.Flush(); err != nil {
			return summary, fmt.Errorf("flush transition jsonl export: %w", err)
		}
		return summary, nil
	case TransitionExportFormatJSON:
		if _, err := bufferedWriter.WriteString("[\n"); err != nil {
			return TransitionExportSummary{}, fmt.Errorf("write transition json prefix: %w", err)
		}
		firstRecord := true
		summary, err := r.StreamTransitions(ctx, query, func(record TrainingTransitionRecord) error {
			if !firstRecord {
				if _, err := bufferedWriter.WriteString(",\n"); err != nil {
					return fmt.Errorf("write transition json separator: %w", err)
				}
			}
			payload, err := json.Marshal(record)
			if err != nil {
				return fmt.Errorf("marshal transition json row: %w", err)
			}
			if _, err := bufferedWriter.Write(payload); err != nil {
				return fmt.Errorf("write transition json row: %w", err)
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
				return summary, fmt.Errorf("write transition json trailing newline: %w", err)
			}
		}
		if _, err := bufferedWriter.WriteString("]\n"); err != nil {
			return summary, fmt.Errorf("write transition json suffix: %w", err)
		}
		if err := bufferedWriter.Flush(); err != nil {
			return summary, fmt.Errorf("flush transition json export: %w", err)
		}
		return summary, nil
	default:
		return TransitionExportSummary{}, fmt.Errorf("unsupported transition export format %q", format)
	}
}

// buildTrainingTransitionsSelectQuery keeps the SQL contract centralized so CLI exporters and
// future trainer readers stay aligned with the trainer-facing transitions schema.
func buildTrainingTransitionsSelectQuery(cfg ClickHouseConfig, query TrainingTransitionQuery) (string, []any) {
	cfg = cfg.normalized()

	var builder strings.Builder
	builder.WriteString("SELECT\n\t")
	builder.WriteString(strings.Join(trainingTransitionSelectColumns, ",\n\t"))
	fmt.Fprintf(&builder, "\nFROM %s.%s_transitions", cfg.Database, cfg.TablePrefix)

	var (
		conditions []string
		args       []any
	)
	appendArgumentCondition := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}

	if query.Scenario != "" {
		appendArgumentCondition("scenario = $%d", query.Scenario)
	}
	if query.Outcome != "" {
		appendArgumentCondition("outcome = $%d", query.Outcome)
	}
	if query.EpisodeIDMin > 0 {
		appendArgumentCondition("episode_id >= $%d", query.EpisodeIDMin)
	}
	if query.EpisodeIDMax > 0 {
		appendArgumentCondition("episode_id <= $%d", query.EpisodeIDMax)
	}
	if len(conditions) > 0 {
		builder.WriteString("\nWHERE ")
		builder.WriteString(strings.Join(conditions, "\n\tAND "))
	}
	builder.WriteString("\nORDER BY episode_id ASC, tick ASC")
	if query.Limit > 0 {
		args = append(args, query.Limit)
		fmt.Fprintf(&builder, "\nLIMIT $%d", len(args))
	}

	return builder.String(), args
}

func normalizedTrainingTransitionQuery(query TrainingTransitionQuery) TrainingTransitionQuery {
	query.Scenario = strings.TrimSpace(query.Scenario)
	query.Outcome = strings.TrimSpace(query.Outcome)
	if query.Limit < 0 {
		query.Limit = 0
	}
	if query.EpisodeIDMax > 0 && query.EpisodeIDMin > query.EpisodeIDMax {
		query.EpisodeIDMin, query.EpisodeIDMax = query.EpisodeIDMax, query.EpisodeIDMin
	}
	return query
}

func normalizedTransitionExportFormat(format TransitionExportFormat) (TransitionExportFormat, error) {
	switch TransitionExportFormat(strings.ToLower(strings.TrimSpace(string(format)))) {
	case "", TransitionExportFormatJSONL:
		return TransitionExportFormatJSONL, nil
	case TransitionExportFormatJSON:
		return TransitionExportFormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported transition export format %q; use jsonl or json", format)
	}
}

// mustTrainingTransitionSelectColumns derives the SQL select list from the documented
// TrainingTransitionRecord contract so any new trainer-facing field is exported automatically
// once the struct grows.
func mustTrainingTransitionSelectColumns() []string {
	recordType := reflect.TypeOf(TrainingTransitionRecord{})
	columns := make([]string, 0, recordType.NumField())
	for index := 0; index < recordType.NumField(); index++ {
		field := recordType.Field(index)
		columnName := strings.TrimSpace(field.Tag.Get("ch"))
		if columnName == "" {
			panic(fmt.Sprintf("training transition field %q is missing ch tag", field.Name))
		}
		columns = append(columns, columnName)
	}
	return columns
}
