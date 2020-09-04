package parser

import (
	"context"
	"encoding/csv"
	"github.com/Gobonoid/glitter-and-gold/internal/mapper"
	"github.com/Gobonoid/glitter-and-gold/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

//CSV behaves as DI container for csv parser
type CSV struct {
	Path       string
	SkipHeader bool
}

//Parse CSV from file path to slice of simplified transactions
func (p *CSV) Parse(ctx context.Context) ([]model.Transaction, error) {
	if err := p.isValid(ctx); err != nil {
		return nil, errors.Wrap(err, "csv is invalid")
	}
	return p.parse(ctx)
}

func (p *CSV) parse(ctx context.Context) ([]model.Transaction, error) {
	csvFile, err := os.Open(p.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %s", p.Path)
	}
	defer func() {
		if err = csvFile.Close(); err != nil {
			log.WithError(err).Warn("failed to close reader, mem leak possible")
		}
	}()

	var t []model.Transaction
	r := csv.NewReader(csvFile)
	if p.SkipHeader {
		_, err = r.Read()
		if err == io.EOF {
			return t, nil
		}
		if err != nil {
			return nil, errors.Wrap(err, "something went terrible wrong while reading CSV header")
		}
	}

	for {
		select {
		case <-ctx.Done():
			return t, nil
		default:
		}
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "something went terrible wrong while reading CSV")
		}

		t = append(t, mapper.MapCSVRecordTransaction(record))

	}
	return t, nil
}

//isValid might be an overhead double read but in case of writing to the DB I want to be sure all rows are fine
func (p *CSV) isValid(ctx context.Context) error {
	csvFile, err := os.Open(p.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", p.Path)
	}
	defer func() {
		if err = csvFile.Close(); err != nil {
			log.WithError(err).Warn("failed to close reader, mem leak possible")
		}
	}()
	r := csv.NewReader(csvFile)
	var i int
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "something went terrible wrong while reading CSV")
		}

		if len(record) != 10 {
			return errors.Errorf("incorrect number of rows at line %s", i)
		}
		i++

		//TODO: ADD extra checks for values, time, etc if this goes anywhere near prod grade code
	}

	return nil
}
