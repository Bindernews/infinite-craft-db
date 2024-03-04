package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"binder.fun/infinicraft/infinidb"
	"binder.fun/infinicraft/infinidb/rdb"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

var optEnvFile = flag.String("env", ".env", ".env file name")
var logger *slog.Logger

func main() {
	var err error

	flag.Parse()
	logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	loadDotEnv()

	dbUrl := Getenv("IFDB_DB_URL", true)

	infi := &infinidb.InfiniDb{
		Log: logger,
		Db:  connectToDb(dbUrl),
	}
	if infi.Db == nil {
		os.Exit(1)
	}
	if err = infi.Setup(); err != nil {
		logger.Error("setup", "error", err)
		os.Exit(1)
	}

	infi.Web.Run("0.0.0.0:8080")
}

func loadDotEnv() {
	// Ignore missing default .env
	if *optEnvFile == ".env" && !isFile(".env") {
		return
	}
	if err := godotenv.Load(*optEnvFile); err != nil {
		logger.Error("dotenv load failed", "path", *optEnvFile, "error", err)
		os.Exit(1)
	}
}

func isFile(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.Mode().IsRegular()
}

func connectToDb(dbUrl string) rdb.RecipeDb {
	if conn, err := pgx.Connect(context.Background(), dbUrl); err != nil {
		logger.Error("database connect failed", "error", err)
		return nil
	} else {
		return rdb.NewRecipeDb(conn)
	}
}

// Utility function to get an environment variable value
func Getenv(key string, fatal bool) string {
	s := os.Getenv(key)
	if s == "" {
		if fatal {
			logger.Error("env missing", "key", key)
			os.Exit(1)
		} else {
			logger.Warn("env missing", "key", key)
		}
	}
	return s
}
