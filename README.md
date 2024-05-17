> [!NOTE]
> You can find the latest developments in the gwctl directory of the Gateway API repository: https://github.com/kubernetes-sigs/gateway-api/tree/main/gwctl"


# gwctl

gwctl is a tool that improves the usability of the Gateway API by providing a better way to view and manage policies ([GEP-713](https://gateway-api.sigs.k8s.io/geps/gep-713)). The aim is to make it available as a standalone binary, a kubectl plugin, and a library.

gwctl allows you to view all Gateway API policy types that are present in a cluster, as well as all "policy bindings" in a namespace (or across all namespaces). It also shows you the attached policies when you view any Gateway resource (like HTTPRoute, Gateway, GatewayClass, etc.)

gwctl uses the `gateway.networking.k8s.io/policy=true` label to identify Policy CRDs (https://gateway-api.sigs.k8s.io/geps/gep-713/#kubectl-plugin)

Please note that gwctl is <b>still under development and may have bugs</b>. There may be changes at various places, including the command-line interface, the output format, and the supported features.

In the future, gwctl may be able to read status from the policy resource to determine if it has been applied correctly.

## Try it out!

```bash
# Clone the gwctl repository
git clone https://github.com/gauravkghildiyal/gwctl.git

# Go to the gwctl directory
cd gwctl

# Ensure vendor depedencies
go mod tidy
go mod vendor

# Build the gwctl binary
go build -o bin/gwctl cmd/gwctl/main.go

# Add binary to PATH
export PATH=./bin:${PATH}

# OPTIONAL: Create sample resources
kubectl apply -f samples/crds.yaml
kubectl apply -f samples/examples.yaml

# Start using!
gwctl --help
```

## Examples
Here are some examples of how gwctl can be used:

```bash
# List all policies in the cluster. This will also give the resource they bind to.
gwctl get policies -A

# List all available policy types
gwctl get policycrds

# Describe all HTTPRoutes in namespace ns2
gwctl describe httproutes -n ns2

# Describe a single HTTPRoute in default namespace
gwctl describe httproutes demo-httproute-1

# Describe all Gateways across all namespaces.
gwctl describe gateways -A

# Describe a single GatewayClass
gwctl describe gatewayclasses foo-com-external-gateway-class
```

Here are some commands with their sample output:
```bash
❯ gwctl get policies -A
POLICYNAME                     POLICYKIND               TARGETNAME                      TARGETKIND
demo-health-check-1            HealthCheckPolicy        demo-gateway-1                  Gateway
demo-retry-policy-1            RetryOnPolicy            demo-gateway-1                  Gateway
demo-retry-policy-2            RetryOnPolicy            demo-httproute-2                HTTPRoute
demo-timeout-policy-1          TimeoutPolicy            foo-com-external-gateway-class  GatewayClass
demo-tls-min-version-policy-1  TLSMinimumVersionPolicy  demo-httproute-1                HTTPRoute
demo-tls-min-version-policy-2  TLSMinimumVersionPolicy  demo-gateway-2                  Gateway

❯ gwctl describe httproutes -n ns2
Name:      demo-httproute-3
Namespace: ns2
Hostnames:
    - example.com
ParentRefs:
    - Kind:  Gateway
      Group: gateway.networking.k8s.io
      Name:  demo-gateway-2
DirectlyAttachedPolicies:
InheritedPolicies:
    - Kind: TLSMinimumVersionPolicy
      Group: baz.com
      Name: demo-tls-min-version-policy-2
      Target:
          Kind:   Gateway
          Group:  gateway.networking.k8s.io
          Name:   demo-gateway-2


Name:      demo-httproute-4
Namespace: ns2
Hostnames:
    - demo.com
ParentRefs:
    - Kind:  Gateway
      Group: gateway.networking.k8s.io
      Name:  demo-gateway-1
DirectlyAttachedPolicies:
InheritedPolicies:
    - Kind: HealthCheckPolicy
      Group: foo.com
      Name: demo-health-check-1
      Target:
          Kind:   Gateway
          Group:  gateway.networking.k8s.io
          Name:   demo-gateway-1
    - Kind: RetryOnPolicy
      Group: foo.com
      Name: demo-retry-policy-1
      Target:
          Kind:   Gateway
          Group:  gateway.networking.k8s.io
          Name:   demo-gateway-1
    - Kind: TimeoutPolicy
      Group: bar.com
      Name: demo-timeout-policy-1
      Target:
          Kind:   GatewayClass
          Group:  gateway.networking.k8s.io
          Name:   foo-com-external-gateway-class
```

---

## Areas that definitely need some work:
* Add tests.
* Add some more tests.
* Re-evalute the minimum information that we need to print for resource descriptions.
* Improve/define library interfaces.
* There's several areas which could be generified instead of being repetitive.

