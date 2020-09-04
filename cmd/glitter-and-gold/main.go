package main

import (
	"context"
	"github.com/Gobonoid/glitter-and-gold/internal/parser"
	"github.com/Gobonoid/glitter-and-gold/internal/processor"
	"github.com/dgraph-io/badger"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

type options struct {
	LogLevel string `default:"info" split_words:"true" description:"Log level [debug|info|warn|error]"`
	CSVPath  string `default:"sample-transactions.csv" description:"Path to csv to be processed"`
}

func main() {
	opts := options{}
	err := envconfig.Process("", &opts)
	if err != nil {
		panic("failed to parse ENV config: " + err.Error())
	}
	configureGlobalLogger(opts.LogLevel)
	p := parser.CSV{
		Path:       opts.CSVPath,
		SkipHeader: true,
	}

	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.WithError(err).Fatal("failed to create tmp DB")
	}
	defer db.Close()
	db.DropAll()
	ctx, cancelFn := context.WithCancel(context.Background())
	var g errgroup.Group
	g.Go(func() error {
		trns, parseErr := p.Parse(ctx)
		if parseErr != nil {
			return errors.Wrap(parseErr, "failed to parse CSV")
		}
		processor.TransactionProcessor{DB: db}.Process(ctx, trns)
		return nil
	})
	//TODO: add monitoring if wanted

	//This is overhead in case this is meant to run as a service or smth...
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	cancelFn()
	if err = g.Wait(); err != nil {
		log.WithError(err).Panic("failed to serve")
	}

}

func configureGlobalLogger(logLevel string) {
	log.SetFormatter(&log.JSONFormatter{})
	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithError(err).Fatal("error parsing log level")
	}
	log.SetLevel(lvl)
}
