package config

import "github.com/jessevdk/go-flags"

type Options struct {
	ConfigFile string `long:"config" default:"config.yml" description:"path to config file"`
	Etcd       bool   `long:"etcd" description:"read config from etcd"`
	Company    string `long:"company" description:"use company subscriber"`
	Shared     bool   `long:"shared" description:"use shared subscriber"`
	Adapter    string `long:"adapter" description:"choose adapter type"`
	Host       string `long:"host" description:"host url"`
	Base       string `long:"base" description:"base path"`
}

func ParseFlags() (*Options, error) {
	var options Options
	parser := flags.NewParser(&options, flags.Default)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}
	return &options, nil
}
