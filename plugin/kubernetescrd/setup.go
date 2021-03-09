package kubernetescrd

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"k8s.io/client-go/tools/clientcmd"
)

const pluginName = "kubernetescrd"

func init() {
	plugin.Register(pluginName, setup)
}

func setup(c *caddy.Controller) error {
	k, err := parseKubernetesCRD(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		k.Next = next
		return k
	})

	return nil
}

func parseKubernetesCRD(c *caddy.Controller) (*KubernetesCRD, error) {
	var (
		k   *KubernetesCRD
		err error
		i   int
	)

	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++
		k, err = parseStanza(c)
		if err != nil {
			return nil, err
		}
	}

	return k, nil
}

func parseStanza(c *caddy.Controller) (*KubernetesCRD, error) {
	k := &KubernetesCRD{}

	zones := c.RemainingArgs()
	if len(zones) != 0 {
		k.Zones = make([]string, len(zones))
		for i := 0; i < len(k.Zones); i++ {
			k.Zones[i] = plugin.Host(zones[i]).Normalize()
		}
	} else {
		k.Zones = make([]string, len(c.ServerBlockKeys))
		for i := 0; i < len(k.Zones); i++ {
			k.Zones[i] = plugin.Host(c.ServerBlockKeys[i]).Normalize()
		}
	}

	for c.NextBlock() {
		switch c.Val() {
		case "endpoint":
			args := c.RemainingArgs()
			if len(args) != 1 {
				return nil, c.ArgErr()
			}
			k.APIServerEndpoint = args[0]
		case "tls":
			args := c.RemainingArgs()
			if len(args) != 3 {
				return nil, c.ArgErr()
			}
			k.APIClientCert, k.APIClientKey, k.APICertAuth = args[0], args[1], args[2]
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
		case "namespace":
			args := c.RemainingArgs()
			if len(args) != 1 {
				return nil, c.ArgErr()
			}
			k.Namespace = args[0]
		default:
			return nil, c.Errf("unknown property '%s'", c.Val())
		}
	}

	return k, nil
}
