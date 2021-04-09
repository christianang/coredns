# forwardcrd

## Name

*forwardcrd* - enables proxying DNS messages to upstream resolvers by reading
CRDs from a Kubernetes cluster

## Description

The *forwardcrd* plugin is used to dynamically configure stub-domains by
reading a `DNSZone` CRD within a Kubernetes cluster.

See [Configuring Private DNS Zones and Upstream Nameservers in
Kubernetes](https://kubernetes.io/blog/2017/04/configuring-private-dns-zones-upstream-nameservers-kubernetes/)
for a description of stub-domains within Kubernetes.

This plugin can only be used once per Server Block.

## Security

This plugin gives users of Kubernetes another avenue of modifying the CoreDNS
server other than the `coredns` configmap. Therefore, it is important that you
limit the RBAC and the `namespace` the plugin reads from to reduce the surface
area a malicious actor can use. Ideally, the level of access to create `DNSZone`
resources is at the same level as the access to the `coredns` configmap.

## Syntax

~~~
forwardcrd [ZONES...]
~~~

With only the plugin specified, the *forwardcrd* plugin will default to the
zone specified in the server's block. It will allow any `DNSZone` resource that
matches or includes the zone as a suffix. If **ZONES** are specified it allows
any zone listed as a suffix.

```
forwardcrd [ZONES...] {
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
  **Note**: It is recommended to limit the namespace (e.g to `kube-system`)
  because this can be potentially misused. It is ideal to keep the level of
  write access similar to the `coredns` configmap in the `kube-system`
  namespace.

## Ready

This plugin reports readiness to the ready plugin. This will happen after it has
synced to the Kubernetes API.

## Examples

Allow `DNSZone` resources to be created for any zone:

~~~ txt
. {
    forwardcrd
}
~~~

Allow `DNSZone` resources to be created for the `.local` zone:


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

Only read `DNSZone` resources from the `kube-system` namespace:

~~~ txt
. {
    forwardcrd {
        namespace kube-system
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

## DNSZone resource

Apply the `DNSZone` CRD to your Kubernetes cluster.

```
kubectl apply -f ./manifests/crds/coredns.io_dnszones.yaml
```

Assuming the **forwardcrd** plugin has been configured to allow `DNSZone`
resources within any `zone`, but must be created in the `kube-system` namespace.
E.g:

~~~ txt
. {
:   forwardcrd {
        namespace kube-system
    }
}
~~~

Also note that the `ClusterRole` for CoreDNS must include:

```yaml
rules:
- apiGroups:
  - coredns.io
  resources:
  - dnszones
  verbs:
  - list
  - watch
```

Create the following `DNSZone` resource to forward `example.local` to the
nameserver `10.100.0.10`.

```yaml
---
apiVersion: coredns.io/v1alpha1
kind: DNSZone
metadata:
  name: example-local
  namespace: kube-system
spec:
  zoneName: example.local
  forwardTo: 10.100.0.10
```
