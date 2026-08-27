# Migration: k8s.io/* → v0.37.0

Updated `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go` (direct) from
`v0.36.2` to `v0.37.0`. `github.com/cert-manager/cert-manager` was raised from
`v1.20.3` to `v1.21.1` as part of the same upgrade — v1.20.x does not build
against client-go v0.37.

`k8s.io/apiextensions-apiserver` (indirect) stays at `v0.36.2`: that is what
cert-manager v1.21.1 requires, and nothing in this module depends on it directly.

## Breaking changes

### `k8s.io/client-go` v0.37

The set of typed clients exposed by `kubernetes.Interface` changed again, and the
signature of `Discovery()` changed. Both broke the hand-written fake clientset in
`service/internal/kubernetes/mock/clientset.go`:

- `SchedulingV1alpha2` was replaced by `SchedulingV1alpha3`
  (package `k8s.io/client-go/kubernetes/typed/scheduling/v1alpha2` → `.../v1alpha3`).
- New clients: `LifecycleV1alpha1`, `StoragemigrationV1` (`StoragemigrationV1beta1` stays).
- `Discovery()` now returns `discovery.DiscoveryInterfaces` (plural) instead of
  `discovery.DiscoveryInterface`. `DiscoveryInterfaces` combines `DiscoveryInterface`
  and the new `DiscoveryInterfaceWithContext`. No implementation change was needed —
  `*fakediscovery.FakeDiscovery` already satisfies both.

### `cert-manager` v1.21

- The deprecated alias `cmmeta.ObjectReference` (`type ObjectReference = IssuerReference`)
  was removed. Use `cmmeta.IssuerReference`.
- Because it was a type *alias*, this is source-compatible for consumers of this
  library: `entity.NewIssuerRef` and `(*IssuerRef).ToIssuerRef` keep the same
  underlying type. No major version bump required.

## Steps per module

1. In the module directory:
   ```bash
   go get k8s.io/api@v0.37.0 k8s.io/apimachinery@v0.37.0 k8s.io/client-go@v0.37.0
   go mod tidy
   ```
   `go get` raises cert-manager to v1.21.1 on its own.

2. In `service/internal/kubernetes/mock/clientset.go`, align the fake with the new
   `kubernetes.Interface`:

   - Replace the scheduling v1alpha2 client with v1alpha3:
     ```go
     // before
     schedulingv1alpha2 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha2"
     func (c *KubeClientset) SchedulingV1alpha2() schedulingv1alpha2.SchedulingV1alpha2Interface { panic("not implemented") }

     // after
     schedulingv1alpha3 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha3"
     func (c *KubeClientset) SchedulingV1alpha3() schedulingv1alpha3.SchedulingV1alpha3Interface { panic("not implemented") }
     ```

   - Add the imports and methods for the new clients:
     ```go
     lifecyclev1alpha1  "k8s.io/client-go/kubernetes/typed/lifecycle/v1alpha1"
     storagemigrationv1 "k8s.io/client-go/kubernetes/typed/storagemigration/v1"

     func (c *KubeClientset) LifecycleV1alpha1() lifecyclev1alpha1.LifecycleV1alpha1Interface { panic("not implemented") }
     func (c *KubeClientset) StoragemigrationV1() storagemigrationv1.StoragemigrationV1Interface { panic("not implemented") }
     ```

   - Widen the return type of `Discovery()` (leave `CmClientset.Discovery()` alone —
     it implements cert-manager's clientset interface, not the k8s one):
     ```go
     // before
     func (c *KubeClientset) Discovery() discovery.DiscoveryInterface { return c.discovery }
     // after
     func (c *KubeClientset) Discovery() discovery.DiscoveryInterfaces { return c.discovery }
     ```

3. Replace `cmmeta.ObjectReference` with `cmmeta.IssuerReference` in
   `entity/certificate.go` and `entity/certificate_test.go`.

4. ```bash
   go build ./...
   go test ./...
   ```

## Downstream impact

- `service/backend.KubernetesInterface` is a public alias for `k8s.Interface`.
  Consumers that pass a real `*kubernetes.Clientset` are unaffected. Consumers with
  their own hand-written implementation of that interface (their own test fake) must
  apply the same three method changes as step 2.
- Through MVS, this release raises every consumer to `client-go v0.37.0` and
  `cert-manager v1.21.1`, whether or not they are ready. Call this out in the
  changelog explicitly, and consider whether `lts/**` branches should receive it.

## Non-blocking test failures

None.
