package lightning

import (
	"log"
	"os"

	"github.com/rs/zerolog"
)

var Log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "Jan 02 15:04:05"}).With().Timestamp().Logger().Level(zerolog.InfoLevel)

func SetupLogs(logLevel zerolog.Level) {
	Log = Log.Level(logLevel)

	Log.WithLevel(logLevel).Msg("Set log level!")

	log.SetFlags(0)
	log.SetOutput(Log)
}
