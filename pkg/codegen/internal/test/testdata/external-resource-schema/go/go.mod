module github.com/pulumi/pulumi/pkg/v2/codegen/internal/test/testdata/external-resource-schema/go

go 1.14

require (
	github.com/pulumi/pulumi-kubernetes/sdk/v2 v2.7.2 // indirect
	// throwing this here so we don't inject circular dependency on pkg
	github.com/pulumi/pulumi-random/sdk/v2 v2.4.1
	github.com/pulumi/pulumi/sdk/v2 v2.14.0
)

replace github.com/pulumi/pulumi-random/sdk/v2 => /Users/vivekl/code/pulumi-random/sdk
