// Copyright 2016-2020, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/v2/backend"
	resourceanalyzer "github.com/pulumi/pulumi/pkg/v2/resource/analyzer"
	"github.com/pulumi/pulumi/sdk/v2/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v2/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v2/go/common/util/archive"
	"github.com/pulumi/pulumi/sdk/v2/go/common/util/result"
	"github.com/pulumi/pulumi/sdk/v2/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v2/nodejs/npm"
)

// PublishOperation publishes a PolicyPack to the backend.
type PublishOperation struct {
	Root       string
	PlugCtx    *plugin.Context
	PolicyPack *workspace.PolicyPackProject
	Scopes     CancellationScopeSource
}

// PolicyPackOperation is used to make various operations against a Policy Pack.
type PolicyPackOperation struct {
	// If nil, the latest version is assumed.
	VersionTag *string
	Scopes     CancellationScopeSource
	Config     map[string]*json.RawMessage
}

// PolicyPackIdentifier captures the information necessary to uniquely identify a policy pack.
type PolicyPackIdentifier struct {
	OrgName    string // The name of the organization that administers the policy pack.
	Name       string // The name of the policy pack.
	VersionTag string // The version of the policy pack. Optional.
	ConsoleURL string // The URL of the console for the service that owns this policy pack.
}

func (id PolicyPackIdentifier) String() string {
	return fmt.Sprintf("%s/%s", id.OrgName, id.Name)
}

func (id PolicyPackIdentifier) URL() string {
	return fmt.Sprintf("%s/%s/policypacks/%s", id.ConsoleURL, id.OrgName, id.Name)
}

func ParsePolicyPackIdentifier(s, currentUser, consoleURL string) (PolicyPackIdentifier, error) {
	split := strings.Split(s, "/")
	var orgName string
	var policyPackName string

	switch len(split) {
	case 2:
		orgName = split[0]
		policyPackName = split[1]
	default:
		return PolicyPackIdentifier{}, errors.Errorf("could not parse policy pack name '%s'; must be of the form "+
			"<org-name>/<policy-pack-name>", s)
	}

	if orgName == "" {
		orgName = currentUser
	}

	return PolicyPackIdentifier{
		OrgName:    orgName,
		Name:       policyPackName,
		ConsoleURL: consoleURL,
	}, nil
}

// PolicyPack is a the Pulumi service implementation of the PolicyPack interface.
type PolicyPack struct {
	id PolicyPackIdentifier
	// b is a pointer to the backend that this PolicyPack belongs to.
	b *Backend
	// cl is the client used to interact with the backend.
	cl backend.PolicyClient
}

func (pack *PolicyPack) ID() PolicyPackIdentifier {
	return pack.id
}

func (pack *PolicyPack) Backend() *Backend {
	return pack.b
}

func (pack *PolicyPack) Publish(ctx context.Context, op PublishOperation) result.Result {

	//
	// Get PolicyPack metadata from the plugin.
	//

	fmt.Println("Obtaining policy metadata from policy plugin")

	abs, err := filepath.Abs(op.PlugCtx.Pwd)
	if err != nil {
		return result.FromError(err)
	}

	analyzer, err := op.PlugCtx.Host.PolicyAnalyzer(tokens.QName(abs), op.PlugCtx.Pwd, nil /*opts*/)
	if err != nil {
		return result.FromError(err)
	}

	analyzerInfo, err := analyzer.GetAnalyzerInfo()
	if err != nil {
		return result.FromError(err)
	}

	// Update the name and version tag from the metadata.
	pack.id.Name = analyzerInfo.Name
	pack.id.VersionTag = analyzerInfo.Version

	fmt.Println("Compressing policy pack")

	var packTarball []byte

	// TODO[pulumi/pulumi#1334]: move to the language plugins so we don't have to hard code here.
	runtime := op.PolicyPack.Runtime.Name()
	if strings.EqualFold(runtime, "nodejs") {
		packTarball, err = npm.Pack(op.PlugCtx.Pwd, os.Stderr)
		if err != nil {
			return result.FromError(
				errors.Wrap(err, "could not publish policies because of error running npm pack"))
		}
	} else if strings.EqualFold(runtime, "python") {
		// npm pack puts all the files in a "package" subdirectory inside the .tgz it produces, so we'll do
		// the same for Python. That way, after unpacking, we can look for the PulumiPolicy.yaml inside the
		// package directory to determine the runtime of the policy pack.
		packTarball, err = archive.TGZ(op.PlugCtx.Pwd, "package", true /*useDefaultExcludes*/)
		if err != nil {
			return result.FromError(
				errors.Wrap(err, "could not publish policies because of error creating the .tgz"))
		}
	} else {
		return result.Errorf(
			"failed to publish policies because PulumiPolicy.yaml specifies an unsupported runtime %s",
			runtime)
	}

	//
	// Publish.
	//

	fmt.Println("Uploading policy pack to Pulumi service")

	publishedVersion, err := pack.cl.PublishPolicyPack(ctx, pack.id.OrgName, analyzerInfo, bytes.NewReader(packTarball))
	if err != nil {
		return result.FromError(err)
	}

	fmt.Printf("\nPermalink: %s/%s\n", pack.id.URL(), publishedVersion)

	return nil
}

func (pack *PolicyPack) Enable(ctx context.Context, policyGroup string, op PolicyPackOperation) error {
	versionTag := ""
	if op.VersionTag != nil {
		versionTag = *op.VersionTag
	}
	return pack.cl.EnablePolicyPack(ctx, pack.id.OrgName, policyGroup, pack.id.Name, versionTag, op.Config)
}

func (pack *PolicyPack) Validate(ctx context.Context, op PolicyPackOperation) error {
	schema, err := pack.cl.GetPolicyPackSchema(ctx, pack.id.OrgName, pack.id.Name, *op.VersionTag)
	if err != nil {
		return err
	}
	err = resourceanalyzer.ValidatePolicyPackConfig(schema.ConfigSchema, op.Config)
	if err != nil {
		return err
	}
	return nil
}

func (pack *PolicyPack) Disable(ctx context.Context, policyGroup string, op PolicyPackOperation) error {
	versionTag := ""
	if op.VersionTag != nil {
		versionTag = *op.VersionTag
	}
	return pack.cl.DisablePolicyPack(ctx, pack.id.OrgName, policyGroup, pack.id.Name, versionTag)
}

func (pack *PolicyPack) Remove(ctx context.Context, op PolicyPackOperation) error {
	versionTag := ""
	if op.VersionTag != nil {
		versionTag = *op.VersionTag
	}
	return pack.cl.DeletePolicyPack(ctx, pack.id.OrgName, pack.id.Name, versionTag)
}
