// Copyright 2016-2024, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/logging"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type constructorConverter struct{}

func (*constructorConverter) Close() error {
	return nil
}

func (*constructorConverter) ConvertState(_ context.Context, _ *plugin.ConvertStateRequest,
) (*plugin.ConvertStateResponse, error) {
	return nil, fmt.Errorf("create mapper: ConvertState not implemented")
}

func (*constructorConverter) ConvertProgram(_ context.Context,
	req *plugin.ConvertProgramRequest,
) (*plugin.ConvertProgramResponse, error) {
	loaderClient, err := schema.NewLoaderClient(req.LoaderTarget)
	if err != nil {
		return nil, fmt.Errorf("creating loader client: %w", err)
	}

	if len(req.Args) < 1 {
		return nil, fmt.Errorf("expecting at least 1 argument for the schema source")
	}

	schemaSource := req.Args[0]
	resourceOrFunctionToken := ""
	requiredPropertiesOnly := false
	skipResources := false
	skipFunctions := false
	for _, arg := range req.Args {
		if arg == "--required-properties-only" {
			requiredPropertiesOnly = true
		}

		if arg == "--skip-resources" {
			skipResources = true
		}

		if arg == "--skip-functions" {
			skipFunctions = true
		}

		if strings.Contains(arg, ":") {
			resourceOrFunctionToken = arg
		}
	}

	loadedPackage, err := loadSchema(schemaSource, loaderClient)
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	var resourceSchema *schema.Resource
	var functionSchema *schema.Function
	if resourceOrFunctionToken != "" {
		for _, r := range loadedPackage.Resources {
			if r.Token == resourceOrFunctionToken {
				resourceSchema = r
				break
			}
		}

		if resourceSchema == nil {
			// only if we couldn't find a resource schema, look for a function schema
			for _, f := range loadedPackage.Functions {
				if f.Token == resourceOrFunctionToken {
					functionSchema = f
					break
				}
			}
		}
	}

	generator := &exampleGenerator{
		indentSize:             0,
		requiredPropertiesOnly: requiredPropertiesOnly,
	}

	var code string
	if resourceSchema != nil {
		code = generator.exampleResource(resourceSchema)
	} else if functionSchema != nil {
		code = generator.exampleInvoke(functionSchema)
	} else {
		if resourceOrFunctionToken != "" {
			return nil, fmt.Errorf("resource or function %q not found", resourceOrFunctionToken)
		}

		// generate all
		code = generator.generateAll(loadedPackage, generateAllOptions{
			includeResources: !skipResources,
			includeFunctions: !skipFunctions,
		})
	}

	path := filepath.Join(req.TargetDirectory, "main.pp")
	if err = os.WriteFile(path, []byte(code), 0o600); err != nil {
		return nil, fmt.Errorf("writing generated code: %w", err)
	}

	return &plugin.ConvertProgramResponse{
		Diagnostics: nil,
	}, nil
}

func bindSchema(spec schema.PackageSpec) (*schema.Package, error) {
	pkg, diags, err := schema.BindSpec(spec, nil)
	if err != nil {
		return nil, err
	}
	if diags.HasErrors() {
		return nil, diags
	}
	return pkg, nil
}

func loadSchema(packageSource string, loader schema.ReferenceLoader) (*schema.Package, error) {
	var spec schema.PackageSpec
	if ext := filepath.Ext(packageSource); ext == ".yaml" || ext == ".yml" {
		f, err := os.ReadFile(packageSource)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(f, &spec)
		if err != nil {
			return nil, err
		}
		return bindSchema(spec)
	} else if ext == ".json" {
		f, err := os.ReadFile(packageSource)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(f, &spec)
		if err != nil {
			return nil, err
		}
		return bindSchema(spec)
	}

	var version *semver.Version
	parts := strings.Split(packageSource, "@")
	if len(parts) == 2 {
		packageSource = parts[0]
		v, err := semver.Parse(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parsing version: %w", err)
		}
		version = &v
	}

	loadedPackage, err := loader.LoadPackage(packageSource, version)
	if err != nil {
		return nil, fmt.Errorf("loading package: %w", err)
	}

	return loadedPackage, nil
}

func main() {

	logging.InitLogging(false, 0, false)

	rc, err := rpcCmd.NewRpcCmd(&rpcCmd.RpcCmdConfig{
		TracingName:  "pulumi-converter-constructor-syntax",
		RootSpanName: "pulumi-converter-constructor-syntax",
	})
	if err != nil {
		cmdutil.Exit(err)
	}

	rc.Run(func(srv *grpc.Server) error {
		pulumirpc.RegisterConverterServer(srv, plugin.NewConverterServer(&constructorConverter{}))
		return nil
	}, func() {})
}
