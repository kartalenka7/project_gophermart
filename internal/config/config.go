package config

import (
	"flag"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
)

var (
	CryptoLen = 10
)

type Config struct {
	Server     string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	Database   string `env:"DATABASE_URI"`
	AccrualSys string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8080/"`
}

var (
	localAddr = "localhost:8080"
	baseURL   = "http://localhost:8080/"
	//database  = "postgres://habruser:habr@localhost:5432/habrdb"
	database = "user=habruser password=habr host=localhost port=5432 dbname=habrdb sslmode=disable"
)

func GetConfig(log *logrus.Logger) (Config, error) {
	var cfg Config
	var cfgFlag Config

	// Парсим переменные окружения
	err := env.Parse(&cfg)
	if err != nil {
		return Config{}, err
	}

	// флаг -a, отвечающий за адрес запуска сервиса
	flag.StringVar(&cfgFlag.Server, "a", localAddr, "HTTP server address")

	flag.StringVar(&cfgFlag.Database, "d", database, "Database connections")

	// флаг -r адрес системы расчета начислений
	flag.StringVar(&cfgFlag.AccrualSys, "r", baseURL, "Accrual system")
	flag.Parse()

	log.WithFields(logrus.Fields{"cfgFlag": cfgFlag}).Info("Получены флаги командной строки")

	if cfg.Server == "" || cfg.Server == localAddr {
		cfg.Server = cfgFlag.Server
	}

	if cfg.Database == "" || cfg.Database == database {
		cfg.Database = cfgFlag.Database
	}

	if cfg.AccrualSys == "" {
		cfg.AccrualSys = cfgFlag.AccrualSys
	}

	log.WithFields(logrus.Fields{"cfg": cfg}).Info("Итоговая конфигурация")
	return cfg, err
}
