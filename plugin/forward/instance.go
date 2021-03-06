package forward

import (
	"crypto/tls"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/parse"
	"github.com/coredns/coredns/plugin/pkg/transport"
)

// All Forward properties get added here.
type ForwardOptions struct {
	From string
	To   []string
	// Except []string
}

// Some thought needs to be put into how this can be made more generic. Maybe
// other plugins may reuse this pattern?
// Instance will end up doing nearly the same thing as parseForward. Perhaps
// parseForward should reuse Instance?
func Instance(options ForwardOptions) (*Forward, error) {
	f := New()
	f.from = plugin.Host(options.From).Normalize()

	if len(options.To) == 0 {
		panic("something bad happened")
	}

	toHosts, err := parse.HostPortOrFile(options.To...)
	if err != nil {
		panic(err)
	}

	transports := make([]string, len(toHosts))
	allowedTrans := map[string]bool{"dns": true, "tls": true}
	for i, host := range toHosts {
		trans, h := parse.Transport(host)

		if !allowedTrans[trans] {
			return f, fmt.Errorf("'%s' is not supported as a destination protocol in forward: %s", trans, host)
		}
		p := NewProxy(h, trans)
		f.proxies = append(f.proxies, p)
		transports[i] = trans
	}

	// configureOptions(options, f)

	if f.tlsServerName != "" {
		f.tlsConfig.ServerName = f.tlsServerName
	}

	// Initialize ClientSessionCache in tls.Config. This may speed up a TLS handshake
	// in upcoming connections to the same TLS server.
	f.tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(len(f.proxies))

	for i := range f.proxies {
		// Only set this for proxies that need it.
		if transports[i] == transport.TLS {
			f.proxies[i].SetTLSConfig(f.tlsConfig)
		}
		f.proxies[i].SetExpire(f.expire)
		f.proxies[i].health.SetRecursionDesired(f.opts.hcRecursionDesired)
	}

	return f, nil
}

// func configureOptions(options ForwardOptions, f *Forward) {
// 	if len(options.Except) > 0 {
// 		ignore := make([]string, len(options.Except))
// 		for i, host := options.Except {
// 			ignore[i] = plugin.Host(host).Normalize()
// 		}
// 		f.ignored = ignore
// 	}

// 	// put the rest here (basically a copy of what parseStanza does)
// }
