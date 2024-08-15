package ch

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

type AnaliticsDB struct {
	l *zap.Logger

	conn     driver.Conn
	pathChan chan *Path
}

type Path struct {
	UnitID       int
	X, Y         float64
	Cost         float64
	GoalX, GoalY float64
}

func Start(l *zap.Logger) (*AnaliticsDB, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"192.168.1.156:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "PrimaBek123",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 600,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:      time.Duration(10) * time.Second,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Duration(10) * time.Minute,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		return nil, err
	}

	ch := AnaliticsDB{
		l:        l,
		conn:     conn,
		pathChan: make(chan *Path, 1000),
	}

	go ch.appendPath()

	return &ch, conn.Ping(context.Background())
}

func (ch *AnaliticsDB) AddPath(s *Path) {
	ch.pathChan <- s
}

func (ch *AnaliticsDB) appendPath() {
	const (
		unitTable = "units"
	)

	t := time.NewTicker(10 * time.Second)
	batch, err := ch.getBatch(unitTable)
	if err != nil {
		ch.l.Error("getBatch err", zap.Error(err))
	}

	var s *Path
	for {
		select {
		case s = <-ch.pathChan:
			if err := batch.Append(
				s.UnitID,
				s.X,
				s.Y,
				s.Cost,
				s.GoalX,
				s.GoalY,
			); err != nil {
				ch.l.Error("Append Batch err", zap.Error(err))
			}
		case <-t.C:
			if err := batch.Send(); err != nil {
				ch.l.Error("Send Batch err", zap.Error(err))
			}
			batch, err = ch.getBatch(unitTable)
			if err != nil {
				ch.l.Error("getBatch err", zap.Error(err))
			}
		}
	}
}

func (ch *AnaliticsDB) getBatch(str string) (driver.Batch, error) {
	batch, err := ch.conn.PrepareBatch(context.Background(), "INSERT INTO "+str)
	if err != nil {
		ch.l.Error("PrepareBatch err", zap.Error(err))
		return nil, err
	}
	return batch, nil
}
