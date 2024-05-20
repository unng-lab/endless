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
	unitChan chan *Unit
}

type Unit struct {
	GoodID   int64
	SearchID int64
	Ts       time.Time
	Position int64

	Ksort int64
	Sort  int64

	Name            string
	BrandID         int64
	SiteBrandID     int64
	SubjectID       int64
	SubjectParentID int64

	Rating    int8
	Feedbacks int32
	Root      int64

	Sale         int8
	SalePriceU   int32
	AveragePrice int32
	Benefit      int16

	Time1 int64
	Time2 int64
}

func Start(l *zap.Logger) (*AnaliticsDB, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"192.168.0.115:9000"},
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
		unitChan: make(chan *Unit, 1000),
	}

	go ch.appendSearches()

	return &ch, conn.Ping(context.Background())
}

func (ch *AnaliticsDB) AddSearches(s *Unit) {
	ch.unitChan <- s
}

func (ch *AnaliticsDB) appendSearches() {
	const (
		unitTable = "units"
	)

	t := time.NewTicker(1 * time.Minute)
	batch, err := ch.getBatch(unitTable)
	if err != nil {
		ch.l.Error("getBatch err", zap.Error(err))
	}

	var s *Unit
	for {
		select {
		case s = <-ch.unitChan:
			if err := batch.Append(
				s.GoodID,
				s.SearchID,
				s.Ts,
				s.Position,

				s.Ksort,
				s.Sort,

				s.Name,
				s.BrandID,
				s.SiteBrandID,
				s.SubjectID,
				s.SubjectParentID,

				s.Rating,
				s.Feedbacks,
				s.Root,

				s.Sale,
				s.SalePriceU,

				s.Time1,
				s.Time2,

				s.Benefit,
				s.AveragePrice,
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
