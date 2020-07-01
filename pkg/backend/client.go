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

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pulumi/pulumi/pkg/v2/engine"
	"github.com/pulumi/pulumi/sdk/v2/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v2/go/common/resource/config"
	"github.com/pulumi/pulumi/sdk/v2/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v2/go/common/workspace"
)

// ListStacksFilter describes optional filters when listing stacks.
type ListStacksFilter struct {
	Project      *string
	Organization *string
	TagName      *string
	TagValue     *string
}

// StackIdentifier is the set of data needed to identify a Pulumi stack.
type StackIdentifier struct {
	Owner   string
	Project string
	Stack   string
}

// ParseStackIdentifier parses the stack name into a backend.StackIdentifier. Any omitted
// portions will be filled in using the given owner and project.
//
// "alpha"            - will just set the Name, but ignore Owner and Project.
// "alpha/beta"       - will set the Owner and Name, but not Project.
// "alpha/beta/gamma" - will set Owner, Name, and Project.
func ParseStackIdentifier(s, defaultOwner, defaultProject string) (StackIdentifier, error) {
	id := StackIdentifier{
		Owner:   defaultOwner,
		Project: defaultProject,
	}

	split := strings.Split(s, "/")
	switch len(split) {
	case 1:
		id.Stack = split[0]
	case 2:
		id.Owner = split[0]
		id.Stack = split[1]
	case 3:
		id.Owner = split[0]
		id.Project = split[1]
		id.Stack = split[2]
	default:
		return StackIdentifier{}, fmt.Errorf("could not parse stack name '%s'", s)
	}

	return id, nil
}

func ParseStackIdentifierWithClient(ctx context.Context, s string, client Client) (StackIdentifier, error) {
	id, err := ParseStackIdentifier(s, "", "")
	if err != nil {
		return StackIdentifier{}, err
	}

	if id.Owner == "" {
		currentUser, err := client.User(ctx)
		if err != nil {
			return StackIdentifier{}, err
		}
		id.Owner = currentUser
	}

	if id.Project == "" {
		currentProject, err := workspace.DetectProject()
		if err != nil {
			return StackIdentifier{}, err
		}
		id.Project = currentProject.Name.String()
	}

	return id, nil
}

func (id StackIdentifier) String() string {
	return fmt.Sprintf("%s/%s/%s", id.Owner, id.Project, id.Stack)
}

func (id StackIdentifier) FriendlyName(currentUser, currentProject string) string {
	// If the project names match or if the stack has no project, we can elide the project name.
	if currentProject == id.Project || id.Project == "" {
		if id.Owner == currentUser || id.Owner == "" {
			return id.Stack // Elide owner too, if it is the current user or if it is the empty string.
		}
		return fmt.Sprintf("%s/%s", id.Owner, id.Stack)
	}

	return fmt.Sprintf("%s/%s/%s", id.Owner, id.Project, id.Stack)
}

// Update tracks an ongoing deployment operation.
type Update interface {
	ProgressURL() string
	PermalinkURL() string

	RecordEvent(ctx context.Context, event apitype.EngineEvent) error
	PatchCheckpoint(ctx context.Context, deployment *apitype.DeploymentV3) error
	Complete(ctx context.Context, status apitype.UpdateStatus) error
}

// Client implements the low-level operations required by the Pulumi CLI.
type Client interface {
	Name() string
	URL() string
	User(ctx context.Context) (string, error)
	DefaultSecretsManager() string

	DoesProjectExist(ctx context.Context, owner, projectName string) (bool, error)
	StackConsoleURL(stackID StackIdentifier) (string, error)

	ListStacks(ctx context.Context, filter ListStacksFilter) ([]apitype.StackSummary, error)
	GetStack(ctx context.Context, stackID StackIdentifier) (apitype.Stack, error)
	CreateStack(ctx context.Context, stackID StackIdentifier, tags map[string]string) (apitype.Stack, error)
	DeleteStack(ctx context.Context, stackID StackIdentifier, force bool) (bool, error)
	RenameStack(ctx context.Context, currentID, newID StackIdentifier) error
	UpdateStackTags(ctx context.Context, stack StackIdentifier, tags map[string]string) error

	GetStackHistory(ctx context.Context, stackID StackIdentifier) ([]apitype.UpdateInfo, error)
	GetLatestStackConfig(ctx context.Context, stackID StackIdentifier) (config.Map, error)
	ExportStackDeployment(ctx context.Context, stackID StackIdentifier, version *int) (apitype.UntypedDeployment, error)
	ImportStackDeployment(ctx context.Context, stackID StackIdentifier, deployment *apitype.UntypedDeployment) error

	StartUpdate(ctx context.Context, kind apitype.UpdateKind, stackID StackIdentifier, proj *workspace.Project,
		cfg config.Map, metadata apitype.UpdateMetadata, opts engine.UpdateOptions, tags map[string]string,
		dryRun bool) (Update, error)
	CancelCurrentUpdate(ctx context.Context, stackID StackIdentifier) error
}

// PolicyClient implements the low-level policy pack operations required by the Pulumi CLI.
type PolicyClient interface {
	ListPolicyGroups(ctx context.Context, orgName string) (apitype.ListPolicyGroupsResponse, error)
	ListPolicyPacks(ctx context.Context, orgName string) (apitype.ListPolicyPacksResponse, error)

	GetPolicyPack(ctx context.Context, location string) ([]byte, error)
	GetPolicyPackSchema(ctx context.Context, orgName, policyPackName,
		versionTag string) (*apitype.GetPolicyPackConfigSchemaResponse, error)

	PublishPolicyPack(ctx context.Context, orgName string, analyzerInfo plugin.AnalyzerInfo,
		dirArchive io.Reader) (string, error)
	DeletePolicyPack(ctx context.Context, orgName, policyPackName, versionTag string) error

	EnablePolicyPack(ctx context.Context, orgName, policyGroup, policyPackName, versionTag string,
		policyPackConfig map[string]*json.RawMessage) error
	DisablePolicyPack(ctx context.Context, orgName, policyGroup, policyPackName, versionTag string) error
}
