package kubernetescrd

import (
	"context"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"k8s.io/client-go/tools/clientcmd"
)

func init() { plugin.Register("kubernetescrd", setup) }

func setup(c *caddy.Controller) error {
	k, err := kubernetescrdParse(c)
	if err != nil {
		panic(err)
	}

	err = k.InitKubeCache(context.Background())
	if err != nil {
		panic(err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		k.Next = next
		return k
	})

	c.OnStartup(func() error {
		go k.APIConn.Run()

		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				if k.APIConn.HasSynced() {
					return nil
				}
			case <-timeout:
				return nil
			}
		}
	})

	c.OnShutdown(func() error {
		return k.APIConn.Stop()
	})

	return nil
}

func kubernetescrdParse(c *caddy.Controller) (*KubernetesCRD, error) {
	var (
		k   *KubernetesCRD
		err error
	)

	i := 0
	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++

		k, err = ParseStanza(c)
		if err != nil {
			return k, err
		}
	}
	return k, nil
}

// ParseStanza parses a kubernetes stanza
func ParseStanza(c *caddy.Controller) (*KubernetesCRD, error) {
	k := &KubernetesCRD{}

	for c.NextBlock() {
		switch c.Val() {
		case "kubeconfig":
			args := c.RemainingArgs()
			if len(args) != 1 && len(args) != 2 {
				return nil, c.ArgErr()
			}
			overrides := &clientcmd.ConfigOverrides{}
			if len(args) == 2 {
				overrides.CurrentContext = args[1]
			}
			config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: args[0]},
				overrides,
			)
			k.ClientConfig = config
		default:
			return nil, c.Errf("unknown property '%s'", c.Val())
		}
	}

	return k, nil
}
