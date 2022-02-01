package common

import "fmt"

var (
	ErrConnEtcd = func(err error) error { return fmt.Errorf("failed to connect etcd; %w", err) }
)
