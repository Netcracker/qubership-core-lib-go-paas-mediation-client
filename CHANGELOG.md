## 8.10.0 — Gateway API HTTPRoute timeouts

### Added
- Envoy Gateway `BackendTrafficPolicy` support: automatically created/updated/deleted alongside `HTTPRoute` to configure `streamIdleTimeout` and GRPC backend protocol (`useClientProtocol`).
- `RouteSpec.StreamIdleTimeout` and `RouteSpec.Filters` fields.
- `PlatformClientBuilder.WithHTTPRouteRequestIdleTimeout()` / `HTTP_ROUTE_REQUEST_IDLE_TIMEOUT` service property for a default idle timeout.
- `entity.IsValidGatewayAPIDuration`, `entity.Route.ToBackendTrafficPolicy`, `entity.RouteFromHTTPRouteWithPolicy`.
- `backend.KubernetesApi.DynamicInterface` for dynamic-client access to `BackendTrafficPolicy`.

### Changed
- `HTTPRoute` no longer carries `nginx.ingress.kubernetes.io/*` annotations or `spec.timeouts.request`; timeout/GRPC config now lives in the companion `BackendTrafficPolicy`.
- Timeout resolution priority: `spec.streamIdleTimeout` → `HTTP_ROUTE_REQUEST_IDLE_TIMEOUT` → legacy nginx annotations.
  When `HTTP_ROUTE_REQUEST_IDLE_TIMEOUT` is empty, `streamIdleTimeout` is derived as the maximum of `nginx.ingress.kubernetes.io/proxy-read-timeout` and `nginx.ingress.kubernetes.io/proxy-send-timeout` (seconds → Gateway API duration). When `HTTP_ROUTE_REQUEST_IDLE_TIMEOUT` is set, it takes precedence over these annotations.

```yaml
  metadata:
    annotations:
      nginx.ingress.kubernetes.io/proxy-read-timeout: "1800"
      nginx.ingress.kubernetes.io/proxy-send-timeout: "900"
      nginx.ingress.kubernetes.io/backend-protocol: "GRPC"   # optional, enables useClientProtocol
  # -> streamIdleTimeout: 1800s (max of read/send), unless overridden by
  #    spec.streamIdleTimeout or HTTP_ROUTE_REQUEST_IDLE_TIMEOUT
```

### Requires
- RBAC: `get/list/watch/create/update/patch/delete` on `gateway.envoyproxy.io/backendtrafficpolicies` (same as existing `HTTPRoute` permissions).
- Envoy Gateway v1.8.0+ — for `BackendTrafficPolicy` v1alpha1 fields used here (`useClientProtocol`, `timeout.http.streamIdleTimeout`, `mergeType: StrategicMerge`).