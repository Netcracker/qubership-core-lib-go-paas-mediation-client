package cache

type CacheName string

const AllCache CacheName = "AllCache"
const CertificateCache CacheName = "CertificateCache"
const ConfigMapCache CacheName = "ConfigMapCache"
const NamespaceCache CacheName = "NamespaceCache"
const RouteCache CacheName = "RouteCache" //todo rename to Ingresses
const SecretCache CacheName = "SecretCache"
const ServiceCache CacheName = "ServiceCache"
const HttpRouteCache CacheName = "HttpRouteCache"
const GrpcRouteCache CacheName = "GrpcRouteCache"
