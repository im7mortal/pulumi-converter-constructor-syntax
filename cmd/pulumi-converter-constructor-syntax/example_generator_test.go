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
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/stretchr/testify/assert"
)

func bindTestSchema(t *testing.T, spec schema.PackageSpec) *schema.Package {
	pkg, diags, err := schema.BindSpec(spec, nil)
	assert.Nil(t, diags)
	assert.Nil(t, err)
	return pkg
}

func TestExampleGeneratorForResource(t *testing.T) {
	t.Parallel()
	pkg := bindTestSchema(t, schema.PackageSpec{
		Name: "test",
		Resources: map[string]schema.ResourceSpec{
			"test:index:MyResource": {
				InputProperties: map[string]schema.PropertySpec{
					"fooString": {
						TypeSpec: schema.TypeSpec{
							Type: "string",
						},
					},
					"fooInt": {
						TypeSpec: schema.TypeSpec{
							Type: "integer",
						},
					},
					"fooBool": {
						TypeSpec: schema.TypeSpec{
							Type: "boolean",
						},
					},
				},
			},
		},
	})

	var resource *schema.Resource
	for _, r := range pkg.Resources {
		if r.Token == "test:index:MyResource" {
			resource = r
			break
		}
	}

	assert.NotNil(t, resource)

	g := exampleGenerator{}
	actual := g.exampleResource(resource)
	expected := `resource "example" "test:index:MyResource" {
  fooBool = false
  fooInt = 0
  fooString = "string"
}`

	assert.Equal(t, expected, actual)
}

func TestExampleGeneratorForAllResource(t *testing.T) {
	t.Parallel()
	pkg := bindTestSchema(t, schema.PackageSpec{
		Name: "test",
		Resources: map[string]schema.ResourceSpec{
			"test:index:Example": {
				InputProperties: map[string]schema.PropertySpec{
					"fooString": {
						TypeSpec: schema.TypeSpec{
							Type: "string",
						},
					},
					"fooInt": {
						TypeSpec: schema.TypeSpec{
							Type: "integer",
						},
					},
					"fooBool": {
						TypeSpec: schema.TypeSpec{
							Type: "boolean",
						},
					},
				},
			},
			"test:index:AnotherExample": {
				InputProperties: map[string]schema.PropertySpec{
					"fooString": {
						TypeSpec: schema.TypeSpec{
							Type: "string",
						},
					},
				},
			},
		},
	})

	g := exampleGenerator{}
	actual := g.generateAll(pkg, generateAllOptions{
		includeResources: true,
	})
	expected := `\\ Creating Resource test:index:AnotherExample
resource "anotherExampleResource" "test:index:AnotherExample" {
  fooString = "string"
}
\\ Creating Resource test:index:Example
resource "exampleResource" "test:index:Example" {
  fooBool = false
  fooInt = 0
  fooString = "string"
}
`

	assert.Equal(t, expected, actual)
}
