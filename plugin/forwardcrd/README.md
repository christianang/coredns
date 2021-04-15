# forwardcrd

## Name

*forwardcrd* - enables proxying DNS messages to upstream resolvers by reading
the `Forward` CRD from a Kubernetes cluster

## Description

The *forwardcrd* plugin is used to dynamically configure stub-domains by
reading a `Forward` CRD within a Kubernetes cluster.

See [Configuring Private DNS Zones and Upstream Nameservers in
Kubernetes](https://kubernetes.io/blog/2017/04/configuring-private-dns-zones-upstream-nameservers-kubernetes/)
for a description of stub-domains within Kubernetes.

This plugin can only be used once per Server Block.

## Security

This plugin gives users of Kubernetes another avenue of modifying the CoreDNS
server other than the `coredns` configmap. Therefore, it is important that you
limit the RBAC and the `namespace` the plugin reads from to reduce the surface
area a malicious actor can use. Ideally, the level of access to create `Forward`
resources is at the same level as the access to the `coredns` configmap.

## Syntax

~~~
forwardcrd [ZONES...]
~~~

With only the plugin specified, the *forwardcrd* plugin will default to the
zone specified in the server's block. It will allow any `Forward` resource that
matches or includes the zone as a suffix. If **ZONES** are specified it allows
any zone listed as a suffix.

```
forwardcrd [ZONES...] {
  endpoint URL
  tls CERT KEY CACERT
  kubeconfig KUBECONFIG [CONTEXT]
  namespace [NAMESPACE]
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
* `namespace` **[NAMESPACE]** only reads `Forward` resources from the namespace
  listed. If this option is omitted then it will read from the default
  namespace, `kube-system`. If this option is specified without any namespaces
  listed it will read from all namespaces.  **Note**: It is recommended to limit
  the namespace (e.g to `kube-system`) because this can be potentially misused.
  It is ideal to keep the level of write access similar to the `coredns`
  configmap in the `kube-system` namespace.

## Ready

This plugin reports readiness to the ready plugin. This will happen after it has
synced to the Kubernetes API.

## Ordering

Forward behavior can be defined in three ways, via a Server Block, via the *forwardcrd* plugin, and via the *forward* plugin.  If more than one of these methods is employed and a query falls within the zone of more than one, CoreDNS selects which one to use based on the following precedence:
Corefile Server Block -> *forwardcrd* plugin -> *forward* plugin.

When `Forward` CRDs and Server Blocks define stub domains that are used,
domains defined in the Corefile take precedence (in the event of zone overlap).
e.g. if the
domain `example.com` is defined in the Corefile as a stub domain, and a
`Forward` CRD record defined for `sub.example.com`, then `sub.example.com` would
get forwarded to the upstream defined in the Corefile, not the `Forward` CRD.

When using `forwardcrd` and `forward` in the same Server Block, `Forward` CRDs
take precedence over the `forward` plugin defined in the same Server Block.
e.g. if a `Forward` resource is defined for `.`, then no queries would be
forwarded to the upstream defined the `forward` plugin of the same Server Block.

## Metrics

`Forward` CRD metrics are all labeled in a single zone (the zone of the enclosing
Server Block).

## Examples

Allow `Forward` resources to be created for any zone and only read `Forward`
resources from the `kube-system` namespace:

~~~ txt
. {
    forwardcrd
}
~~~

Allow `Forward` resources to be created for the `.local` zone and only read
`Forward` resources from the `kube-system` namespace:


~~~ txt
. {
    forwardcrd local
}
~~~

or:

~~~ txt
local {
    forwardcrd
}
~~~

Only read `Forward` resources from the `dns-system` namespace:

~~~ txt
. {
    forwardcrd {
        namespace dns-system
    }
}
~~~

Read `Forward` resources from all namespaces:

~~~ txt
. {
    forwardcrd {
        namespace
    }
}
~~~

Connect to Kubernetes with CoreDNS running outside the cluster:

~~~ txt
. {
    forwardcrd {
        endpoint https://k8s-endpoint:8443
        tls cert key cacert
    }
}
~~~

or:

~~~ txt
. {
    forwardcrd {
        kubeconfig ./kubeconfig
    }
}
~~~

## Forward resource

Apply the `Forward` CRD to your Kubernetes cluster.

```
kubectl apply -f ./manifests/crds/coredns.io_forwards.yaml
```

Assuming the **forwardcrd** plugin has been configured to allow `Forward`
resources in the `kube-system` namespace within any `zone`.
E.g:

~~~ txt
. {
    forwardcrd
}
~~~

Also note that the `ClusterRole` for CoreDNS must include:

```yaml
rules:
- apiGroups:
  - coredns.io
  resources:
  - forwards
  verbs:
  - list
  - watch
```

Create the following `Forward` resource to forward `example.local` to the
nameserver `10.100.0.10`.

```yaml
---
apiVersion: coredns.io/v1alpha1
kind: Forward
metadata:
  name: example-local
  namespace: kube-system
spec:
  from: example.local
  to:
  - 10.100.0.10
```
