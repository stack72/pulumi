// *** WARNING: this file was generated by test. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Threading.Tasks;
using Pulumi.Serialization;

namespace Pulumi.Example
{
    [ResourceIdentifier("example::Cat", "0.0.1")]
    public partial class Cat : Pulumi.CustomResource
    {
        [Output("name")]
        public Output<string?> Name { get; private set; } = null!;


        /// <summary>
        /// Create a Cat resource with the given unique name, arguments, and options.
        /// </summary>
        ///
        /// <param name="name">The unique name of the resource</param>
        /// <param name="args">The arguments used to populate this resource's properties</param>
        /// <param name="options">A bag of options that control this resource's behavior</param>
        public Cat(string name, CatArgs? args = null, CustomResourceOptions? options = null)
            : base("example::Cat", name, args ?? new CatArgs(), MakeResourceOptions(options, ""))
        {
        }

        private Cat(string name, Input<string> id, CustomResourceOptions? options = null)
            : base("example::Cat", name, null, MakeResourceOptions(options, id))
        {
        }

        private static CustomResourceOptions MakeResourceOptions(CustomResourceOptions? options, Input<string>? id)
        {
            var defaultOptions = new CustomResourceOptions
            {
                Version = Utilities.Version,
            };
            var merged = CustomResourceOptions.Merge(defaultOptions, options);
            // Override the ID if one was specified for consistency with other language SDKs.
            merged.Id = id ?? merged.Id;
            return merged;
        }
        /// <summary>
        /// Get an existing Cat resource's state with the given name, ID, and optional extra
        /// properties used to qualify the lookup.
        /// </summary>
        ///
        /// <param name="name">The unique name of the resulting resource.</param>
        /// <param name="id">The unique provider ID of the resource to lookup.</param>
        /// <param name="options">A bag of options that control this resource's behavior</param>
        public static Cat Get(string name, Input<string> id, CustomResourceOptions? options = null)
        {
            return new Cat(name, id, options);
        }
    }

    public sealed class CatArgs : Pulumi.ResourceArgs
    {
        [Input("age")]
        public Input<int>? Age { get; set; }

        [Input("pet")]
        public Input<Inputs.PetArgs>? Pet { get; set; }

        public CatArgs()
        {
        }
    }
}
