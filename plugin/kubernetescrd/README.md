# kubernetescrd

## Name

*kubernetescrd* - enables proxying DNS messages to upstream resolvers by reading
CRDs from a Kubernetes cluster

## Description

The *kubernetescrd* plugin is used to dynamically configure stub-domains by
reading a `DNSZone` CRD within a Kubernetes cluster.

See [Configuring Private DNS Zones and Upstream Nameservers in
Kubernetes](https://kubernetes.io/blog/2017/04/configuring-private-dns-zones-upstream-nameservers-kubernetes/)
for a description of stub-domains within Kubernetes.

This plugin can only be used once per Server Block.

## Syntax

~~~
kubernetescrd [ZONES...]
~~~

With only the plugin specified, the *kubernetescrd* plugin will default to the
zone specified in the server's block. It will allow any `DNSZone` resource that
matches or includes the zone as a suffix. If **ZONES** are specified it allows
any zone listed as a suffix.

```
kubernetescrd [ZONES...] {
  endpoint URL
  tls CERT KEY CACERT
  kubeconfig KUBECONFIG [CONTEXT]
  namespace NAMESPACE
}
```

* `endpoint` specifies the **URL** for a remote k8s API endpoint.  If omitted,
  it will connect to k8s in-cluster using the cluster service account.
* `tls` **CERT** **KEY** **CACERT** are the TLS cert, key and the CA cert file
  names for remote k8s connection.  This option is ignored if connecting
  in-cluster (i.e. endpoint is not specified).
* `kubeconfig` **KUBECONFIG [CONTEXT]** authenticates the connection to a remote
  k8s cluster using a kubeconfig file.  **[CONTEXT]** is optional, if not set,
  then the current context specified in kubeconfig will be used.  It supports
  TLS, username and password, or token-based authentication.  This option is
  ignored if connecting in-cluster (i.e., the endpoint is not specified).
* `namespace` **NAMESPACE** only reads `DNSZone` resources from the namespace
  listed. If this option is omitted then it will read from all namespaces.

## Ready

This plugin reports readiness to the ready plugin. This will happen after it has
synced to the Kubernetes API.

## Examples

Allow `DNSZone` resources to be created for any zone:

~~~ corefile
. {
    kubernetescrd
}
~~~

Allow `DNSZone` resources to be created for the `.local` zone:


~~~ corefile
. {
    kubernetescrd local
}
~~~

or:

~~~ corefile
local {
    kubernetescrd
}
~~~

Only read `DNSZone` resources from the `kube-system` namespace:

~~~ corefile
. {
    kubernetescrd {
        namespace kube-system
    }
}
~~~

Connect to Kubernetes with CoreDNS running outside the cluster:

~~~ corefile
. {
    kubernetescrd {
        endpoint https://k8s-endpoint:8443
        tls cert key cacert
    }
}
~~~

or:

~~~ corefile
. {
    kubernetescrd {
        kubeconfig ./kubeconfig
    }
}
~~~

## DNSZone resource

Apply the `DNSZone` CRD to your Kubernetes cluster.

```
kubectl apply -f ./manifests/crds/coredns.io_dnszones.yaml
```

Assuming the **kubernetescrd** plugin has been configured to allow `DNSZone`
resources within any `zone`, but must be created in the `kube-system` namespace.
E.g:

~~~ corefile
. {
    kubernetescrd {
        namespace kube-system
    }
}
~~~

Create the following `DNSZone` resource to forward `example.local` to the
nameserver `10.100.0.10`.

```yaml
---
apiVersion: coredns.io/v1alpha1
kind: DNSZone
metadata:
  name: example-local
spec:
  zoneName: example.local
  forwardTo: 10.100.0.10
```
