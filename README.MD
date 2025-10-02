[![Go build](https://github.com/Netcracker/qubership-core-lib-go-paas-mediation-client/actions/workflows/go-build.yml/badge.svg)](https://github.com/Netcracker/qubership-core-lib-go-paas-mediation-client/actions/workflows/go-build.yml)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?metric=coverage&project=Netcracker_qubership-core-lib-go-paas-mediation-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-paas-mediation-client)
[![duplicated_lines_density](https://sonarcloud.io/api/project_badges/measure?metric=duplicated_lines_density&project=Netcracker_qubership-core-lib-go-paas-mediation-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-paas-mediation-client)
[![vulnerabilities](https://sonarcloud.io/api/project_badges/measure?metric=vulnerabilities&project=Netcracker_qubership-core-lib-go-paas-mediation-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-paas-mediation-client)
[![bugs](https://sonarcloud.io/api/project_badges/measure?metric=bugs&project=Netcracker_qubership-core-lib-go-paas-mediation-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-paas-mediation-client)
[![code_smells](https://sonarcloud.io/api/project_badges/measure?metric=code_smells&project=Netcracker_qubership-core-lib-go-paas-mediation-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-paas-mediation-client)

Golang implementation of paas agnostic access to the resources of Openshift/Kubernetes

<!-- TOC -->
      * [Supported operations:](#supported-operations)
      * [Usage:](#usage)
      * [Create client with all type of caches enabled](#create-client-with-all-type-of-caches-enabled)
      * [Create client with cache for Service and ConfigMap only](#create-client-with-cache-for-service-and-configmap-only)
      * [Create client with custom rollout executor](#create-client-with-custom-rollout-executor)
        * [Custom rollout parallelism level](#custom-rollout-parallelism-level)
        * [Run rollout in parallel](#run-rollout-in-parallel)
        * [Run rollout sequentially](#run-rollout-sequentially)
      * [Blue Green detection mechanism for Ingress creation](#blue-green-detection-mechanism-for-ingress-creation)
<!-- TOC -->

#### Supported operations:
* CRUD operations on resources
* Watch API on resources changes

#### Usage:

 Requirements:
 1. Register security implementation - dummy or your own, the following example shows registration of required services:
    ```go
    import (
        "github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
        "github.com/netcracker/qubership-core-lib-go/v3/security"
    )

    func init() {
        serviceloader.Register(1, &security.DummyToken{})
    }
    ```
 2. create client with the code: service.NewPlatformClientBuilder()
 3. Optionally turn on required cache
 4. invoke Build() method

#### Create client with all type of caches enabled

```
service.NewPlatformClientBuilder().WithAllCaches().Build()
```

#### Create client with cache for Service and ConfigMap only

```
service.NewPlatformClientBuilder().
		WithServiceCache().
		WithConfigMapCache().
		Build()
```

#### Create client with custom rollout executor

The library allows rolling out deployments both sequentially and parallel. For parallel execution the library uses executor of fixed goroutines.
This allows to control the max amount of simultaneous rollouts the library can perform at any given time.
By default, the value is 32, but custom value can be specified during client creation.

##### Custom rollout parallelism level

```
service.NewPlatformClientBuilder().
		WithAllCaches().
		WithRolloutParallelism(100).
		Build()
```

##### Run rollout in parallel
```
client, _ := service.NewPlatformClientBuilder().
		WithAllCaches().
		Build()

ctx := context.Background()
var deploymentNames []string
platformService.RolloutDeploymentsInParallel(ctx, namespace, deploymentNames)
```

##### Run rollout sequentially
```
client, _ := service.NewPlatformClientBuilder().
		WithAllCaches().
		Build()

ctx := context.Background()
var deploymentNames []string
platformService.RolloutDeployments(ctx, namespace, deploymentNames)
```

#### Blue Green detection mechanism for Ingress creation
Blue Green 2.0 requires Ingresses to be created with ingressClass 'bg.mesh.netcracker.com'
paas-mediation-client will detect BlueGreen mode automatically in the following cases:

1. You created client providing manually consul enabled, consul url and consul token
    ```
    client, err := service.NewPlatformClientBuilder().
        WithConsul(true, "http://consul:8500", "<consul-token>").
        Build()
    ```

2. You created client providing manually consul enabled, consul url only assuming consul token will be acquired automatically via M2M token
    ```
    client, err := service.NewPlatformClientBuilder().
        WithConsul(true, "http://consul:8500").
        Build()
    ```

3. You created client assuming consul enabled, consul url and consul token params will be acquired automatically via envs or properties
consul.enabled prop / CONSUL_ENABLED env
consul.url prop / CONSUL_URL env
consul.token prop / CONSUL_TOKEN env
    ```
    client, err := service.NewPlatformClientBuilder().Build()
    ```
