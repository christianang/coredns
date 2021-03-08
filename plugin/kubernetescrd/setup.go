package kubernetescrd

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register("kubernetescrd", setup)
}

func setup(c *caddy.Controller) error {
	return nil
}
