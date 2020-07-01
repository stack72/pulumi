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

package pulumi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/v2/backend/pulumi/client"
	"github.com/pulumi/pulumi/pkg/v2/engine"
	"github.com/pulumi/pulumi/sdk/v2/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v2/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v2/go/common/util/archive"
	"github.com/pulumi/pulumi/sdk/v2/go/common/util/contract"
	"github.com/pulumi/pulumi/sdk/v2/go/common/util/logging"
	"github.com/pulumi/pulumi/sdk/v2/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v2/nodejs/npm"
	"github.com/pulumi/pulumi/sdk/v2/python"
)

type RequiredPolicy struct {
	apitype.RequiredPolicy
	client  *client.Client
	orgName string
}

var _ engine.RequiredPolicy = (*RequiredPolicy)(nil)

func newRequiredPolicy(client *client.Client,
	policy apitype.RequiredPolicy, orgName string) *RequiredPolicy {

	return &RequiredPolicy{
		client:         client,
		RequiredPolicy: policy,
		orgName:        orgName,
	}
}

func (rp *RequiredPolicy) Name() string    { return rp.RequiredPolicy.Name }
func (rp *RequiredPolicy) Version() string { return strconv.Itoa(rp.RequiredPolicy.Version) }
func (rp *RequiredPolicy) OrgName() string { return rp.orgName }

func (rp *RequiredPolicy) Install(ctx context.Context) (string, error) {
	policy := rp.RequiredPolicy

	// If version tag is empty, we use the version tag. This is to support older version of
	// pulumi/policy that do not have a version tag.
	version := policy.VersionTag
	if version == "" {
		version = strconv.Itoa(policy.Version)
	}
	policyPackPath, installed, err := workspace.GetPolicyPath(rp.OrgName(),
		strings.Replace(policy.Name, tokens.QNameDelimiter, "_", -1), version)
	if err != nil {
		// Failed to get a sensible PolicyPack path.
		return "", err
	} else if installed {
		// We've already downloaded and installed the PolicyPack. Return.
		return policyPackPath, nil
	}

	fmt.Printf("Installing policy pack %s %s...\n", policy.Name, version)

	// PolicyPack has not been downloaded and installed. Do this now.
	policyPackTarball, err := rp.client.DownloadPolicyPack(ctx, policy.PackLocation)
	if err != nil {
		return "", err
	}

	return policyPackPath, installRequiredPolicy(policyPackPath, policyPackTarball)
}

func (rp *RequiredPolicy) Config() map[string]*json.RawMessage { return rp.RequiredPolicy.Config }

const packageDir = "package"

func installRequiredPolicy(finalDir string, tarball []byte) error {
	// If part of the directory tree is missing, ioutil.TempDir will return an error, so make sure
	// the path we're going to create the temporary folder in actually exists.
	if err := os.MkdirAll(filepath.Dir(finalDir), 0700); err != nil {
		return errors.Wrap(err, "creating plugin root")
	}

	tempDir, err := ioutil.TempDir(filepath.Dir(finalDir), fmt.Sprintf("%s.tmp", filepath.Base(finalDir)))
	if err != nil {
		return errors.Wrapf(err, "creating plugin directory %s", tempDir)
	}

	// The policy pack files are actually in a directory called `package`.
	tempPackageDir := filepath.Join(tempDir, packageDir)
	if err := os.MkdirAll(tempPackageDir, 0700); err != nil {
		return errors.Wrap(err, "creating plugin root")
	}

	// If we early out of this function, try to remove the temp folder we created.
	defer func() {
		contract.IgnoreError(os.RemoveAll(tempDir))
	}()

	// Uncompress the policy pack.
	err = archive.UnTGZ(tarball, tempDir)
	if err != nil {
		return err
	}

	logging.V(7).Infof("Unpacking policy pack %q %q\n", tempDir, finalDir)

	// If two calls to `plugin install` for the same plugin are racing, the second one will be
	// unable to rename the directory. That's OK, just ignore the error. The temp directory created
	// as part of the install will be cleaned up when we exit by the defer above.
	if err := os.Rename(tempPackageDir, finalDir); err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "moving plugin")
	}

	projPath := filepath.Join(finalDir, "PulumiPolicy.yaml")
	proj, err := workspace.LoadPolicyPack(projPath)
	if err != nil {
		return errors.Wrapf(err, "failed to load policy project at %s", finalDir)
	}

	// TODO[pulumi/pulumi#1334]: move to the language plugins so we don't have to hard code here.
	if strings.EqualFold(proj.Runtime.Name(), "nodejs") {
		if err := completeNodeJSInstall(finalDir); err != nil {
			return err
		}
	} else if strings.EqualFold(proj.Runtime.Name(), "python") {
		if err := completePythonInstall(finalDir, projPath, proj); err != nil {
			return err
		}
	}

	fmt.Println("Finished installing policy pack")
	fmt.Println()

	return nil
}

func completeNodeJSInstall(finalDir string) error {
	if bin, err := npm.Install(finalDir, nil, os.Stderr); err != nil {
		return errors.Wrapf(
			err,
			"failed to install dependencies of policy pack; you may need to re-run `%s install` "+
				"in %q before this policy pack works", bin, finalDir)
	}

	return nil
}

func completePythonInstall(finalDir, projPath string, proj *workspace.PolicyPackProject) error {
	const venvDir = "venv"
	if err := python.InstallDependencies(finalDir, venvDir, false /*showOutput*/); err != nil {
		return err
	}

	// Save project with venv info.
	proj.Runtime.SetOption("virtualenv", venvDir)
	if err := proj.Save(projPath); err != nil {
		return errors.Wrapf(err, "saving project at %s", projPath)
	}

	return nil
}
