package main

import (
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/stretchr/testify/assert"
	"testing"
)

func bindTestSchema(t *testing.T, spec schema.PackageSpec) *schema.Package {
	pkg, diags, err := schema.BindSpec(spec, nil)
	assert.Nil(t, diags)
	assert.Nil(t, err)
	return pkg
}

func TestExampleGeneratorForResource(t *testing.T) {
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
