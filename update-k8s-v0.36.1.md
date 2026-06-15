# Migration: k8s.io/* → v0.36.1

Updated `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go` (direct) and
`k8s.io/apiextensions-apiserver` (indirect) from `v0.35.x` to `v0.36.1`.

## Breaking changes

`k8s.io/client-go` v0.36 changed the set of typed clients exposed by
`kubernetes.Interface`, which broke the hand-written fake clientset in
`service/internal/kubernetes/mock/clientset.go`:

- Removed clients (no replacement): `AutoscalingV2beta1`, `AutoscalingV2beta2`.
- `SchedulingV1alpha1` was replaced by `SchedulingV1alpha2`
  (package `k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1` → `.../v1alpha2`).

## Steps per module

1. In the module directory:
   ```bash
   go get k8s.io/api@v0.36.1 k8s.io/apimachinery@v0.36.1 k8s.io/client-go@v0.36.1 k8s.io/apiextensions-apiserver@v0.36.1
   go mod tidy
   ```

2. In `service/internal/kubernetes/mock/clientset.go`, align the fake with the new
   `kubernetes.Interface`:

   - Remove the imports and methods for the dropped autoscaling clients:
     ```go
     // removed imports
     autoscalingv2beta1 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"
     autoscalingv2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
     // removed methods
     func (c *KubeClientset) AutoscalingV2beta1() ... { panic("not implemented") }
     func (c *KubeClientset) AutoscalingV2beta2() ... { panic("not implemented") }
     ```

   - Replace the scheduling v1alpha1 client with v1alpha2:
     ```go
     // before
     schedulingv1alpha1 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"
     func (c *KubeClientset) SchedulingV1alpha1() schedulingv1alpha1.SchedulingV1alpha1Interface { panic("not implemented") }

     // after
     schedulingv1alpha2 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha2"
     func (c *KubeClientset) SchedulingV1alpha2() schedulingv1alpha2.SchedulingV1alpha2Interface { panic("not implemented") }
     ```

3. ```bash
   go build ./...
   go test ./...
   ```

## Non-blocking test failures

None.
