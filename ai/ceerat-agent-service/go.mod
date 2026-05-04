module github.com/kaansari/ceerat-platform/ai/ceerat-agent-service

go 1.26.2

require (
	github.com/kaansari/ceerat-platform/packages/ceerat-contracts v0.0.0
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.6
)

require (
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
)

replace github.com/kaansari/ceerat-platform/packages/ceerat-contracts => ../../packages/ceerat-contracts
