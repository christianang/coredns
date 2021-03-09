package kubernetescrd

import (
	"strings"
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
)

func TestKubernetesCRDParse(t *testing.T) {
	c := caddy.NewTestController("dns", `kubernetescrd`)
	_, err := parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		endpoint http://localhost:9090
	}`)
	k, err := parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}
	if k.APIServerEndpoint != "http://localhost:9090" {
		t.Errorf("Expected APIServerEndpoint to be: %s\n but was: %s\n", "http://localhost:9090", k.APIServerEndpoint)
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		tls cert.crt key.key cacert.crt
	}`)
	k, err = parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}
	if k.APIClientCert != "cert.crt" {
		t.Errorf("Expected APIClientCert to be: %s\n but was: %s\n", "cert.crt", k.APIClientCert)
	}
	if k.APIClientKey != "key.key" {
		t.Errorf("Expected APIClientCert to be: %s\n but was: %s\n", "key.key", k.APIClientKey)
	}
	if k.APICertAuth != "cacert.crt" {
		t.Errorf("Expected APICertAuth to be: %s\n but was: %s\n", "cacert.crt", k.APICertAuth)
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		kubeconfig foo.kubeconfig
	}`)
	_, err = parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		kubeconfig foo.kubeconfig context
	}`)
	_, err = parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `kubernetescrd example.org`)
	k, err = parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}
	if len(k.Zones) != 1 || k.Zones[0] != "example.org." {
		t.Fatalf("Expected Zones to consist of \"example.org.\" but was %v", k.Zones)
	}

	c = caddy.NewTestController("dns", `kubernetescrd`)
	c.ServerBlockKeys = []string{"example.org"}
	k, err = parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}
	if len(k.Zones) != 1 || k.Zones[0] != "example.org." {
		t.Fatalf("Expected Zones to consist of \"example.org.\" but was %v", k.Zones)
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		namespace kube-system
	}`)
	k, err = parseKubernetesCRD(c)
	if err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}
	if k.Namespace != "kube-system" {
		t.Errorf("Expected Namespace to be: %s\n but was: %s\n", "kube-system", k.Namespace)
	}

	// negative

	c = caddy.NewTestController("dns", `kubernetescrd {
		endpoint http://localhost:9090 http://foo.bar:1024
	}`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), "Wrong argument count") {
		t.Fatalf("Expected error containing \"Wrong argument count\", but got: %v", err.Error())
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		endpoint
	}`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), "Wrong argument count") {
		t.Fatalf("Expected error containing \"Wrong argument count\", but got: %v", err.Error())
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		tls foo bar
	}`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), "Wrong argument count") {
		t.Fatalf("Expected error containing \"Wrong argument count\", but got: %v", err.Error())
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		kubeconfig
	}`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), "Wrong argument count") {
		t.Fatalf("Expected error containing \"Wrong argument count\", but got: %v", err.Error())
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		kubeconfig too many args
	}`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), "Wrong argument count") {
		t.Fatalf("Expected error containing \"Wrong argument count\", but got: %v", err.Error())
	}

	c = caddy.NewTestController("dns", `kubernetescrd {
		invalid
	}`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), "unknown property") {
		t.Fatalf("Expected error containing \"unknown property\", but got: %v", err.Error())
	}

	c = caddy.NewTestController("dns", `kubernetescrd
kubernetescrd`)
	_, err = parseKubernetesCRD(c)
	if err == nil {
		t.Fatalf("Expected errors, but got nil")
	}
	if !strings.Contains(err.Error(), plugin.ErrOnce.Error()) {
		t.Fatalf("Expected error containing \"%s\", but got: %v", plugin.ErrOnce.Error(), err.Error())
	}
}
