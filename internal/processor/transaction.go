package processor

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/Gobonoid/glitter-and-gold/internal/model"
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sort"
	"strings"
)

type BalanceStore interface {
	UpdateUserMonthlyBalanceByTransaction(trn model.Transaction) error
}

//TransactionProcessor implements parser TransactionProcessor and behaves as DI
type TransactionProcessor struct {
	DB *badger.DB
}

//Process slice of transactions to display required criteria
func (t TransactionProcessor) Process(ctx context.Context, transactions []model.Transaction) error {
	if err := t.aggregateMonthlyBalance(ctx, transactions); err != nil {
		return errors.Wrap(err, "failed to wrap")
	}

	if err := t.displayBestSpenders(); err != nil {
		return errors.Wrap(err, "something went wrong with displaying best spenders")
	}

	return nil
}

func (t TransactionProcessor) aggregateMonthlyBalance(ctx context.Context, transactions []model.Transaction) error {
	//TODO: At this point I should introduce store abstraction but to I am too lazy
	if err := t.DB.Update(func(txn *badger.Txn) error {
		for _, trn := range transactions {
			if !trn.IsGoldSpend() {
				continue
			}
			var balance int64
			v, err := txn.Get(trn.MonthlyBalanceKey())
			if err != nil {
				if err == badger.ErrKeyNotFound {
					balance = 0
				} else {
					return errors.Wrap(err, "failed to get monthly balance")
				}
			} else {
				if v == nil {
					return errors.Wrap(err, "unexpected nil balance in the DB")
				}
				if err = v.Value(func(val []byte) error {
					b := bytes.NewReader(val)
					if readErr := binary.Read(b, binary.LittleEndian, &balance); err != nil {
						return errors.Wrap(readErr, "failed to read binary result into buffer")
					}
					return nil
				}); err != nil {
					return errors.Wrap(err, "failed to read value")
				}
			}
			if setErr := txn.Set(trn.MonthlyBalanceKey(), trn.UpdateRunningBalanceByAmount(balance)); setErr != nil {
				return errors.Wrap(setErr, "failed to set monthly balance")
			}
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "failed to update set")
	}

	return nil
}

func (t TransactionProcessor) displayBestSpenders() error {
	if err := t.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		opts.Reverse = true

		it := txn.NewIterator(opts)
		defer it.Close()

		var spend []userSpend
		var currentPeriod string

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				pence := bytesToPence(v)
				period, email := explodeKey(k)
				if period != currentPeriod {
					currentPeriod = period
					if len(spend) > 0 {
						sort.Slice(spend, func(i, j int) bool {
							return spend[i].amount > spend[j].amount
						})
						fmt.Printf("3 first elements: %d\n", spend[:3])
					}
					spend = []userSpend{}
					return nil
				}
				spend = append(spend, userSpend{email: email, amount: pence})
				fmt.Printf("key=%s, value=%d\n", k, pence)
				return nil
			})
			if err != nil {
				return err
			}
		}
		if len(spend) > 0 {
			sort.Slice(spend, func(i, j int) bool {
				return spend[i].amount > spend[j].amount
			})
			fmt.Printf("3 first elements: %d\n", spend[:3])
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "failed to view DB data")
	}
	return nil
}

func bytesToPence(v []byte) int64 {
	nowBuffer := bytes.NewReader(v)
	var nowVar int64
	if err := binary.Read(nowBuffer, binary.LittleEndian, &nowVar); err != nil {
		log.WithError(err).Panic("failed to read buffer")
	}
	return nowVar
}

func explodeKey(k []byte) (string, string) {
	v := strings.Split(string(k), ";")
	return v[0] + v[1], v[2]
}

type userSpend struct {
	email  string
	amount int64
}
