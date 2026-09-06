package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env string   `yaml:"env" env:"ENV" env-default:"local"`
	Srv Server   `yaml:"server"`
	DB  DataBase `yaml:"db"`
}

type Server struct {
	Host string `yaml:"host" env:"SERVER_HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
}

type DataBase struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER_NAME" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name" env:"DB_NAME" env-default:"postgres"`
	Sslmode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`
}

func ConvertToStringDSN(dbCfg DataBase) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.DBName, dbCfg.Sslmode,
	)
}

func MustLoad() (*Config, error) {
	var cfg Config

	path := fetchConfigPath()
	if path == "" {
		return nil, fmt.Errorf("failed to read config path")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %s", err.Error())
	}

	err = validator(cfg)
	if err != nil {
		return nil, fmt.Errorf("incorrect data config: %s", err.Error())
	}

	return &cfg, nil
}

func fetchConfigPath() string {
	var path string

	flag.StringVar(&path, "config_path", "", "path to config")

	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	return path
}

func validator(cfg Config) error {
	if cfg.Srv.Port < 0 {
		return fmt.Errorf("server port must be a positive number")
	}
	if cfg.DB.Port < 0 {
		return fmt.Errorf("db port must be a positive number")
	}
	if cfg.DB.Password == "" {
		return fmt.Errorf("password must not be empty")

	}
	return nil
}
