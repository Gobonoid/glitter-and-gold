package mapper

import (
	"github.com/Gobonoid/glitter-and-gold/internal/model"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	csvDateTimeLayout = "02/01/2006 15:04"
)

//MapCSVRecordTransaction defined in the model pkg
func MapCSVRecordTransaction(r []string) model.Transaction {
	return model.Transaction{
		FirstName:    r[0],
		LastName:     r[1],
		Email:        r[2],
		Description:  r[3],
		MerchantCode: r[4],
		Amount: func() float64 {
			v, err := strconv.ParseFloat(r[5], 64)
			if err != nil {
				log.WithError(err).Panic("failed to parse amount")
			}
			return v
		}(),
		FromCurrency: r[6],
		ToCurrency:   r[7],
		Rate:         r[8],
		DateTime: func() time.Time {
			t, err := time.Parse(csvDateTimeLayout, r[9])
			if err != nil {
				log.WithError(err).WithField("record", r).Panic("failed to parse dateTime from CSV")
			}
			return t
		}(),
	}
}
