package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

//Transaction is simple representation of data provided by GLINT
type Transaction struct {
	FirstName    string
	LastName     string
	Email        string
	Description  string
	MerchantCode string
	Amount       float64
	FromCurrency string
	ToCurrency   string
	Rate         string
	DateTime     time.Time
}

const (
	goldCurrencyCode = "GGM"
	monthlyBalanceKeyFormat = "%d;%d;%s;balance"
)

//MonthlyBalanceKey for that transaction to be associated with
func (t Transaction) MonthlyBalanceKey() []byte {
	return []byte(fmt.Sprintf(
		monthlyBalanceKeyFormat, t.DateTime.Year(), t.DateTime.Month(), t.Email))
}

func (t Transaction) UpdateRunningBalanceByAmount(balance int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, balance + t.amountInPence())
	if err != nil {
		log.WithError(err).Panic("failed to write into buffer")
	}
	return buf.Bytes()
}

func(t Transaction) IsGoldSpend() bool {
	return t.MerchantCode != "" && t.ToCurrency == goldCurrencyCode
}

func(t Transaction) amountInPence() int64 {
	return int64(math.RoundToEven(t.Amount * 100))
}