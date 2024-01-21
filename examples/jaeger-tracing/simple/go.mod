module github.com/envoyproxy/envoy/examples/jaeger-tracing/simple

// the version should >= 1.18
go 1.20

// NOTICE: these lines could be generated automatically by "go mod tidy"
require (
	github.com/cncf/xds/go v0.0.0-20230607035331-e9ce68804cb4
	github.com/envoyproxy/envoy v1.24.0
	github.com/openzipkin/zipkin-go v0.4.2
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/envoyproxy/protoc-gen-validate v0.10.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230525234035-dd9d682886f9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230525234030-28d5490b6b19 // indirect
	google.golang.org/grpc v1.57.0 // indirect
)

// TODO: remove when #26173 lands.
// And check the "API compatibility" section in doc:
// https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/golang_filter#developing-a-go-plugin
replace github.com/envoyproxy/envoy => ../../..
