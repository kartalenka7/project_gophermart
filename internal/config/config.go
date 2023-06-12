package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env"
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

func GetConfig() (Config, error) {
	var cfg Config
	var cfgFlag Config

	// Парсим переменные окружения
	err := env.Parse(&cfg)
	if err != nil {
		return Config{}, err
	}

	log.Println(cfgFlag)
	// флаг -a, отвечающий за адрес запуска сервиса
	flag.StringVar(&cfgFlag.Server, "a", localAddr, "HTTP server address")

	flag.StringVar(&cfgFlag.Database, "d", database, "Database connections")

	// флаг -r адрес системы расчета начислений
	flag.StringVar(&cfgFlag.AccrualSys, "r", baseURL, "Accrual system")
	flag.Parse()

	log.Printf("Флаги командной строки: %s\n", cfgFlag)
	log.Printf("Переменные конфигурации: %s\n", &cfg)

	if cfg.Server == "" || cfg.Server == localAddr {
		cfg.Server = cfgFlag.Server
	}

	if cfg.Database == "" || cfg.Database == database {
		cfg.Database = cfgFlag.Database
	}

	if cfg.AccrualSys == "" {
		cfg.AccrualSys = cfgFlag.AccrualSys
	}
	return cfg, err
}
