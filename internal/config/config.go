package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string            `yaml:"env" env-default:"local"`
	DSN         string            `yaml:"dsn" env-required:"true"`
	TokenTTL    time.Duration     `yaml:"token_ttl" env-default:"1h"`
	HTTP        HTTPConfig        `yaml:"http"`
	FileStorage FileStorageConfig `yaml:"file_storage"`
	Redis       RedisConf         `yaml:"redis"`
}

type HTTPConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port" env-default:"8080"`
}

type FileStorageConfig struct {
	BaseDir string `yaml:"base_dir"`
	BaseURL string `yaml:"base_url"`
	MaxSize int64  `yaml:"max_size"`
}

type RedisConf struct {
	RedisAddr     string `yaml:"redis_addr"`
	RedisPassword string `yaml:"redispassword"`
	RedisDB       int    `yaml:"redis_db"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config path is empty")
	}

	return MustLoadPath(path)
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	// --config="path/to/config.yaml"
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
