// *** WARNING: this file was generated by test. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

package example

import (
	"context"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

type Cat struct {
	pulumi.CustomResourceState

	Name pulumi.StringPtrOutput `pulumi:"name"`
}

// NewCat registers a new resource with the given unique name, arguments, and options.
func NewCat(ctx *pulumi.Context,
	name string, args *CatArgs, opts ...pulumi.ResourceOption) (*Cat, error) {
	if args == nil {
		args = &CatArgs{}
	}

	var resource Cat
	err := ctx.RegisterResource("example::Cat", name, args, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetCat gets an existing Cat resource's state with the given name, ID, and optional
// state properties that are used to uniquely qualify the lookup (nil if not required).
func GetCat(ctx *pulumi.Context,
	name string, id pulumi.IDInput, state *CatState, opts ...pulumi.ResourceOption) (*Cat, error) {
	var resource Cat
	err := ctx.ReadResource("example::Cat", name, id, state, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// Input properties used for looking up and filtering Cat resources.
type catState struct {
	Name *string `pulumi:"name"`
}

type CatState struct {
	Name pulumi.StringPtrInput
}

func (CatState) ElementType() reflect.Type {
	return reflect.TypeOf((*catState)(nil)).Elem()
}

type catArgs struct {
	Age *int `pulumi:"age"`
	Pet *Pet `pulumi:"pet"`
}

// The set of arguments for constructing a Cat resource.
type CatArgs struct {
	Age pulumi.IntPtrInput
	Pet PetPtrInput
}

func (CatArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*catArgs)(nil)).Elem()
}

type CatInput interface {
	pulumi.Input

	ToCatOutput() CatOutput
	ToCatOutputWithContext(ctx context.Context) CatOutput
}

func (Cat) ElementType() reflect.Type {
	return reflect.TypeOf((*Cat)(nil)).Elem()
}

func (i Cat) ToCatOutput() CatOutput {
	return i.ToCatOutputWithContext(context.Background())
}

func (i Cat) ToCatOutputWithContext(ctx context.Context) CatOutput {
	return pulumi.ToOutputWithContext(ctx, i).(CatOutput)
}

type CatOutput struct {
	*pulumi.OutputState
}

func (CatOutput) ElementType() reflect.Type {
	return reflect.TypeOf((*CatOutput)(nil)).Elem()
}

func (o CatOutput) ToCatOutput() CatOutput {
	return o
}

func (o CatOutput) ToCatOutputWithContext(ctx context.Context) CatOutput {
	return o
}

func init() {
	pulumi.RegisterOutputType(CatOutput{})
}
