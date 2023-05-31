package main

import (
	"fmt"

	"github.com/OdyseeTeam/odysee-api/apps/forklift"
	"github.com/OdyseeTeam/odysee-api/pkg/configng"
	"github.com/OdyseeTeam/odysee-api/pkg/logging"
	"github.com/OdyseeTeam/odysee-api/pkg/logging/zapadapter"
	"github.com/OdyseeTeam/odysee-api/pkg/migrator"

	"github.com/alecthomas/kong"
)

var cli struct {
	Serve struct{} `cmd:"" help:"Start forklift service"`
	Debug bool     `help:"Enable verbose logging"`
}

type loggingConfig struct {
	level, format string
}

func main() {
	ctx := kong.Parse(&cli)

	logCfg := loggingConfig{}
	if cli.Debug {
		logCfg.format = "console"
		logCfg.level = logging.LevelDebug
	} else {
		logCfg.format = "json"
		logCfg.level = logging.LevelDebug
	}
	logger := zapadapter.NewKV(nil)

	switch ctx.Command() {
	case "serve":
		serve(logger)
	default:
		logger.Fatal("unknown command", "name", ctx.Command())
	}
}

func serve(logger logging.KVLogger) {
	cfg, err := configng.Read("./config", "forklift", "yaml")
	if err != nil {
		panic(err)
	}

	s3cfg, err := cfg.ReadS3Config("IncomingStorage")
	if err != nil {
		panic(fmt.Errorf("cannot parse s3 config: %w", err))
	}

	client, err := configng.NewS3ClientV2(s3cfg)
	if err != nil {
		panic(fmt.Errorf("cannot create s3 client: %w", err))
	}

	pgcfg := cfg.ReadPostgresConfig("Database")
	db, err := migrator.ConnectDB(pgcfg)
	if err != nil {
		logger.Fatal("db connection failed", "err", err)
	}

	l := forklift.NewLauncher(
		forklift.WithLogger(logger),
		forklift.WithReflectorConfig(cfg.V.GetStringMapString("ReflectorStorage")),
		forklift.WithConcurrency(cfg.V.GetInt("Concurrency")),
		forklift.WithBlobPath(cfg.V.GetString("BlobPath")),
		forklift.WithRetriever(forklift.NewS3Retriever(cfg.V.GetString("UploadPath"), client)),
		forklift.WithRedisURL(cfg.V.GetString("RedisBus")),
		forklift.WithDB(db),
	)

	b, err := l.Build()
	if err != nil {
		panic(err)
	}
	b.StartHandlers()
}
