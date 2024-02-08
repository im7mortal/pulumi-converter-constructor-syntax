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
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi/pkg/v3/codegen"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
)

type exampleGenerator struct {
	indentSize             int
	requiredPropertiesOnly bool
}

func (g *exampleGenerator) indented(f func()) {
	g.indentSize += 2
	f()
	g.indentSize -= 2
}

func (g *exampleGenerator) indent(buffer *bytes.Buffer) {
	buffer.WriteString(strings.Repeat(" ", g.indentSize))
}

func (g *exampleGenerator) write(buffer *bytes.Buffer, format string, args ...interface{}) {
	buffer.WriteString(fmt.Sprintf(format, args...))
}

func (g *exampleGenerator) writeValue(
	buffer *bytes.Buffer,
	valueType schema.Type,
	seenTypes codegen.StringSet,
) {
	write := func(format string, args ...interface{}) {
		g.write(buffer, format, args...)
	}

	writeValue := func(valueType schema.Type) {
		g.writeValue(buffer, valueType, seenTypes)
	}

	switch valueType {
	case schema.AnyType:
		write("\"any\"")
	case schema.JSONType:
		write("\"{}\"")
	case schema.BoolType:
		write("false")
	case schema.IntType:
		write("0")
	case schema.NumberType:
		write("0.0")
	case schema.StringType:
		write("\"string\"")
	case schema.ArchiveType:
		write("fileArchive(\"./path/to/archive\")")
	case schema.AssetType:
		write("stringAsset(\"content\")")
	}

	switch valueType := valueType.(type) {
	case *schema.ArrayType:
		write("[")
		writeValue(valueType.ElementType)
		write("]")
	case *schema.MapType:
		write("{\n")
		g.indented(func() {
			g.indent(buffer)
			write("\"string\" = ")
			writeValue(valueType.ElementType)
			write("\n")
		})
		g.indent(buffer)
		write("}")
	case *schema.ObjectType:
		if seenTypes.Has(valueType.Token) && objectTypeHasRecursiveReference(valueType) {
			write("notImplemented(%q)", valueType.Token)
			return
		}

		seenTypes.Add(valueType.Token)
		write("{\n")
		g.indented(func() {
			sortPropertiesByRequiredFirst(valueType.Properties)
			for _, p := range valueType.Properties {
				if p.DeprecationMessage != "" {
					continue
				}

				if g.requiredPropertiesOnly && !p.IsRequired() {
					continue
				}

				g.indent(buffer)
				write("%s = ", p.Name)
				writeValue(p.Type)
				write("\n")
			}
		})
		g.indent(buffer)
		write("}")
	case *schema.ResourceType:
		write("notImplemented(%q)", valueType.Token)
	case *schema.EnumType:
		cases := make([]string, len(valueType.Elements))
		for index, c := range valueType.Elements {
			if c.DeprecationMessage != "" {
				continue
			}

			if stringCase, ok := c.Value.(string); ok && stringCase != "" {
				cases[index] = stringCase
			} else if intCase, ok := c.Value.(int); ok {
				cases[index] = strconv.Itoa(intCase)
			} else {
				if c.Name != "" {
					cases[index] = c.Name
				}
			}
		}

		if len(cases) > 0 {
			write(fmt.Sprintf("%q", cases[0]))
		} else {
			write("null")
		}
	case *schema.UnionType:
		if isUnionOfObjects(valueType) && len(valueType.ElementTypes) >= 1 {
			writeValue(valueType.ElementTypes[0])
		}

		for _, elem := range valueType.ElementTypes {
			if isPrimitiveType(elem) {
				writeValue(elem)
				return
			}
		}
		write("null")

	case *schema.InputType:
		writeValue(valueType.ElementType)
	case *schema.OptionalType:
		writeValue(valueType.ElementType)
	case *schema.TokenType:
		writeValue(valueType.UnderlyingType)
	}
}

func (g *exampleGenerator) exampleResource(r *schema.Resource) string {
	buffer := bytes.Buffer{}
	seenTypes := codegen.NewStringSet()
	g.write(&buffer, "resource \"example\" %q {\n", r.Token)
	g.indented(func() {
		sortPropertiesByRequiredFirst(r.InputProperties)
		for _, p := range r.InputProperties {
			if p.DeprecationMessage != "" {
				continue
			}

			if g.requiredPropertiesOnly && !p.IsRequired() {
				continue
			}

			g.indent(&buffer)
			g.write(&buffer, "%s = ", p.Name)
			g.writeValue(&buffer, codegen.ResolvedType(p.Type), seenTypes)
			g.write(&buffer, "\n")
		}
	})

	g.write(&buffer, "}")
	return buffer.String()
}

func (g *exampleGenerator) exampleInvoke(function *schema.Function) string {
	buffer := bytes.Buffer{}
	seenTypes := codegen.NewStringSet()
	g.write(&buffer, "example = invoke(\"%s\", {\n", function.Token)
	g.indented(func() {
		if function.Inputs == nil {
			return
		}

		sortPropertiesByRequiredFirst(function.Inputs.Properties)
		for _, p := range function.Inputs.Properties {
			if p.DeprecationMessage != "" {
				continue
			}

			if g.requiredPropertiesOnly && !p.IsRequired() {
				continue
			}

			g.indent(&buffer)
			g.write(&buffer, "%s = ", p.Name)
			g.writeValue(&buffer, codegen.ResolvedType(p.Type), seenTypes)
			g.write(&buffer, "\n")
		}
	})

	g.write(&buffer, "})")
	return buffer.String()
}

func sortPropertiesByRequiredFirst(props []*schema.Property) {
	sort.Slice(props, func(i, j int) bool {
		return props[i].IsRequired() && !props[j].IsRequired()
	})
}

func isPrimitiveType(t schema.Type) bool {
	switch t {
	case schema.BoolType, schema.IntType, schema.NumberType, schema.StringType:
		return true
	default:
		switch argType := t.(type) {
		case *schema.OptionalType:
			return isPrimitiveType(argType.ElementType)
		case *schema.EnumType, *schema.ResourceType:
			return true
		}
		return false
	}
}

func isUnionOfObjects(schemaType *schema.UnionType) bool {
	for _, elementType := range schemaType.ElementTypes {
		if _, isObjectType := elementType.(*schema.ObjectType); !isObjectType {
			return false
		}
	}

	return true
}

func objectTypeHasRecursiveReference(objectType *schema.ObjectType) bool {
	isRecursive := false
	codegen.VisitTypeClosure(objectType.Properties, func(t schema.Type) {
		if objectRef, ok := t.(*schema.ObjectType); ok {
			if objectRef.Token == objectType.Token {
				isRecursive = true
			}
		}
	})

	return isRecursive
}
