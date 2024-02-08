# Pulumi Constructor Syntax Converter

A Pulumi converter plugin that generates constructor syntax for resources and function invokes in different languages. The inputs of the resources and functions are filled in with default values. Made to be used with the `pulumi convert` command and generate example code that constructs resources and functions for the Pulumi documentation and internal usage.

### Install
`pulumi-converter-constructor-syntax` can be installed using Pulumi's plugin system:
```console
pulumi plugin install converter constructor-syntax
```

## Usage
```
pulumi convert --from constructor-syntax --language <language> --out <output-dir> -- <pkg> <token>
```
Where 
 - `<language>` is the target language of the conversion
 - `<output-dir>` is the directory where the converted code will be written
 - `<pkg>` the source of the schema, it can be a plugin name like `aws`, `aws@<version>` or a path to a schema file
 - `<token>` the token of the resource or function invoke to generate the code for

Optionally another converter argument that can be provided is `--required-properties-only` which will only include the required properties in the constructor syntax.
