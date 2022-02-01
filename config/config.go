package config

import (
	"context"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
	"handago/common"
	"log"
	"time"
)

type Config struct {
	Nats *Nats `yaml:"nats"`
	Etcd *Etcd `yaml:"etcd"`
}

type Nats struct {
	Servers  []string      `yaml:"servers"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
}

type Etcd struct {
	Endpoints []string `yaml:"endpoints" env:"HANDAGO_ETCD_ENDPOINTS"`
	Username  string   `yaml:"username" env:"HANDAGO_ETCD_USERNAME"`
	Password  string   `yaml:"password" env:"HANDAGO_ETCD_PASSWORD"`
	ConfigKey string   `yaml:"-" env:"HANDAGO_ETCD_CONFIG_KEY"`
}

func NewConfig(opts *Options) (*Config, error) {
	var cfg Config

	if opts.Etcd {
		if err := fromEtcd(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	if err := fromFile(&cfg, opts.ConfigFile); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func fromEtcd(cfg *Config) error {
	log.Println("read config from etcd")
	var etcdCfg Etcd
	if err := cleanenv.ReadEnv(&etcdCfg); err != nil {
		return fmt.Errorf("failed to read env variables; %w", err)
	}

	var cli *clientv3.Client
	var err error
	if etcdCfg.Username != "" {
		cli, err = clientv3.New(clientv3.Config{
			Endpoints:   etcdCfg.Endpoints,
			DialTimeout: common.DefaultTimeout,
			Username:    etcdCfg.Username,
			Password:    etcdCfg.Password,
		})
	} else {
		cli, err = clientv3.New(clientv3.Config{
			Endpoints:   etcdCfg.Endpoints,
			DialTimeout: common.DefaultTimeout,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to connect etcd; %w", err)
	}
	defer func() {
		if err = cli.Close(); err != nil {
			log.Printf("failed to close etcd client cleany; %v\n", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTimeout)
	defer cancel()

	resp, err := cli.Get(ctx, etcdCfg.ConfigKey)
	if err != nil {
		return fmt.Errorf("failed to get kv; %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("failed to find value from key: %s", etcdCfg.ConfigKey)
	}

	if err := yaml.Unmarshal(resp.Kvs[0].Value, cfg); err != nil {
		return fmt.Errorf("failed to unmarshal value; %w", err)
	}
	return nil
}

func fromFile(cfg *Config, configPath string) error {
	log.Println("read config from file")
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to read config; %w", err)
	}
	return nil
}
