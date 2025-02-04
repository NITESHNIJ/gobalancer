package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig    `mapstructure:"server"   validate:"required"`
	Pools    []PoolConfig    `mapstructure:"pools"    validate:"required,min=1,dive"`
	Admin    AdminConfig     `mapstructure:"admin"`
	Metrics  MetricsConfig   `mapstructure:"metrics"`
	Logging  LoggingConfig   `mapstructure:"logging"`
}

type ServerConfig struct {
	Addr         string        `mapstructure:"addr"          validate:"required"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	TLS          *TLSConfig    `mapstructure:"tls"`
}

type TLSConfig struct {
	CertFile   string `mapstructure:"cert_file"   validate:"required"`
	KeyFile    string `mapstructure:"key_file"    validate:"required"`
	CAFile     string `mapstructure:"ca_file"`
	Passthrough bool  `mapstructure:"passthrough"`
	MTLS       bool  `mapstructure:"mtls"`
}

type PoolConfig struct {
	Name      string          `mapstructure:"name"      validate:"required"`
	Algorithm string          `mapstructure:"algorithm" validate:"required,oneof=round_robin weighted_round_robin least_connections ip_hash consistent_hash p2c"`
	Backends  []BackendConfig `mapstructure:"backends"  validate:"required,min=1,dive"`
	Routes    []RouteConfig   `mapstructure:"routes"`
	Health    HealthConfig    `mapstructure:"health"`
}

type BackendConfig struct {
	ID     string `mapstructure:"id"     validate:"required"`
	Addr   string `mapstructure:"addr"   validate:"required,url"`
	Weight int    `mapstructure:"weight" validate:"min=0"`
}

type RouteConfig struct {
	PathPrefix string            `mapstructure:"path_prefix"`
	Host       string            `mapstructure:"host"`
	Headers    map[string]string `mapstructure:"headers"`
	Weight     float64           `mapstructure:"weight" validate:"min=0,max=1"`
	Shadow     bool              `mapstructure:"shadow"`
}

type HealthConfig struct {
	Interval           time.Duration `mapstructure:"interval"`
	Timeout            time.Duration `mapstructure:"timeout"`
	Path               string        `mapstructure:"path"`
	FailureThreshold   int           `mapstructure:"failure_threshold"`
	SuccessThreshold   int           `mapstructure:"success_threshold"`
	Mode               string        `mapstructure:"mode" validate:"omitempty,oneof=http tcp grpc"`
	PassiveWindow      time.Duration `mapstructure:"passive_window"`
	PassiveErrorRate   float64       `mapstructure:"passive_error_rate" validate:"min=0,max=1"`
}

type AdminConfig struct {
	Addr string `mapstructure:"addr"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"  validate:"omitempty,oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"omitempty,oneof=json text"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("GOBALANCER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("admin.addr", ":9001")
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

func validate(cfg *Config) error {
	v := validator.New()
	if err := v.Struct(cfg); err != nil {
		var errs []string
		for _, e := range err.(validator.ValidationErrors) {
			errs = append(errs, fmt.Sprintf("field %s: %s", e.Field(), e.Tag()))
		}
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
