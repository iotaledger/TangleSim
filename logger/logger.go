package logger

import (
	"github.com/iotaledger/hive.go/configuration"
	"github.com/iotaledger/hive.go/logger"
)

var New = logger.NewLogger

func init() {
	if err := logger.InitGlobalLogger(configuration.New()); err != nil {
		panic(err)
	}
	logger.SetLevel(logger.LevelDebug)
}
