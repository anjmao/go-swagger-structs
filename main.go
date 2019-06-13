package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/anjmao/go-swagger-structs/swagger"
)

var (
	outDir  = flag.String("output", "./models", "Output file (Eg. ./models)")
	std     = flag.Bool("std", false, "Write to Stdout instead of file")
	source  = flag.String("source", "swagger.json", "Input source as a filename or http url")
	help    = flag.Bool("help", false, "Show help")
	version = flag.Bool("version", false, "Print version")
)

var Version = "0.0.3"

const (
	outFileName = "models.go"
	usageStr    = `
Usage: go-swagger-structs [options]
Options:
	--output <path>				Write models to path (e.g --output ./here/models).
	--std					Print to Stdout instead of file.
	--source <url>				Swagger source url. Could be local file or http(s) url.
Other options:
	--help                      Print help
	--version                   Print version
`
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *help {
		fmt.Printf("%s\n", usageStr)
		os.Exit(0)
	}

	spec, err := fetchSpec(*source)
	if err != nil {
		log.Fatal(err)
	}

	mapper := &modelsMapper{defs: spec.Definitions}

	// Execute template and write output to in memory buffer.
	buf := bytes.NewBuffer([]byte{})
	if err := execTemplate(mapper.getTemplateModel(), buf); err != nil {
		log.Fatal(err)
	}

	// Format output bytes with gofmt.
	outBytes, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	// Write output.
	if *std {
		if err := writeToStdout(outBytes); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := writeToFile(outBytes, *outDir); err != nil {
			log.Fatal(err)
		}
	}
}

type modelsMapper struct {
	defs swagger.Definitions
}

func (m *modelsMapper) toGoPublicFieldName(name string) string {
	if name == "id" {
		return "ID"
	}
	if strings.HasSuffix(name, "Id") {
		name = name[0:len(name)-2] + "ID"
	} else if strings.HasSuffix(name, "Url") {
		name = name[0:len(name)-3] + "URL"
	}
	r, n := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(r)) + name[n:]
}

func (m *modelsMapper) findDefName(ref string) string {
	refName := strings.Replace(ref, "#/definitions/", "", 1)
	if _, ok := m.defs[refName]; ok {
		return refName
	}
	return "unknown"
}

func (m *modelsMapper) toFieldType(prop *swagger.Property, required bool) string {
	prefix := ""
	if required {
		prefix = "*"
	}

	switch prop.Type {
	case swagger.TypeString:
		switch prop.Format {
		case swagger.FormatDateTime:
			return fmt.Sprintf("%s%s", prefix, "time.Time")
		default:
			return fmt.Sprintf("%s%s", prefix, "string")
		}
	case swagger.TypeBoolean:
		return fmt.Sprintf("%s%s", prefix, "bool")
	case swagger.TypeNumber:
		return fmt.Sprintf("%s%s", prefix, "float64")
	case swagger.TypeInteger:
		switch prop.Format {
		case swagger.FormatInt32:
			return fmt.Sprintf("%s%s", prefix, "int32")
		case swagger.FormatInt64:
			return fmt.Sprintf("%s%s", prefix, "int64")
		}
	case swagger.TypeArray:
		return fmt.Sprintf("[]%s", m.toFieldType(prop.Items, false))
	}

	defName := m.findDefName(prop.Ref)
	return fmt.Sprintf("*%s", m.toGoModelName(defName))
}

func (m *modelsMapper) toGoModelFields(requiredFields []string, props swagger.Properties) []*goModelField {
	var out []*goModelField

	// Sort map keys to ensure ordered access.
	var propKeys []string
	for key := range props {
		propKeys = append(propKeys, key)
	}
	sort.Strings(propKeys)

	for _, key := range propKeys {
		prop := props[key]
		typ := m.toFieldType(prop, m.isRequiredField(requiredFields, key))
		field := &goModelField{
			Name:             m.toGoPublicFieldName(key),
			TypeName:         typ,
			Tags:             fmt.Sprintf("`json:\"%s,omitempty\"`", key),
			swaggerFieldName: key,
		}
		out = append(out, field)
	}

	return out
}

func (m *modelsMapper) isRequiredField(requiredFields []string, fieldName string) bool {
	for _, f := range requiredFields {
		if f == fieldName {
			return true
		}
	}
	return false
}

func (m *modelsMapper) toGoModelName(name string) string {
	return strings.Replace(name, ".", "", -1)
}

func (m *modelsMapper) goModels() []*goModel {
	var out []*goModel

	// Sort map keys to ensure ordered access.
	var defKeys []string
	for key := range m.defs {
		defKeys = append(defKeys, key)
	}
	sort.Strings(defKeys)

	for _, key := range defKeys {
		def := m.defs[key]
		fields := m.toGoModelFields(def.Required, def.Properties)
		model := &goModel{
			Name:   m.toGoModelName(key),
			Fields: fields,
		}
		out = append(out, model)
	}

	return out
}

func (m *modelsMapper) getImports() []string {
	imports := map[string]struct{}{}
	for _, v := range m.defs {
		for _, p := range v.Properties {
			if p.Type == swagger.TypeString && p.Format == swagger.FormatDateTime {
				imports["time"] = struct{}{}
			}
		}
	}
	var out []string
	for imp := range imports {
		out = append(out, imp)
	}
	return out
}

func (m *modelsMapper) getTemplateModel() *tmplModel {
	return &tmplModel{
		Models:  m.goModels(),
		Imports: m.getImports(),
	}
}

type goModelField struct {
	Name             string
	TypeName         string
	Tags             string
	swaggerFieldName string
}

type goModel struct {
	Name   string
	Fields []*goModelField
}

type tmplModel struct {
	Imports []string
	Models  []*goModel
}

func execTemplate(model *tmplModel, writer io.Writer) error {
	msgTemplate := `// Code generated by go-swagger-structs. DO NOT EDIT.
package models

{{range .Imports}}
import "{{.}}"
{{- end}}

{{range .Models}}
type {{.Name}} struct {
{{- range .Fields}}
	{{.Name}} {{.TypeName}} {{.Tags}}
{{- end}}
}
{{end}}
`
	tmpl, err := template.New("models").Parse(msgTemplate)
	if err != nil {
		panic(err)
	}

	return tmpl.Execute(writer, model)
}

func writeToFile(data []byte, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return fmt.Errorf("could not create output directory: %v", err)
		}
	}
	if err := ioutil.WriteFile(filepath.Join(path, outFileName), data, os.ModePerm); err != nil {
		return fmt.Errorf("could not write output file: %v", err)
	}
	return nil
}

func writeToStdout(data []byte) error {
	_, err := fmt.Fprint(os.Stdout, string(data))
	return err
}

func fetchSpec(source string) (*swagger.Spec, error) {
	var spec *swagger.Spec
	var err error
	if strings.HasPrefix(source, "http") {
		spec, err := swagger.FetchRemoteSpec(source)
		return spec, err
	} else {
		spec, err = swagger.FetchLocalSpec(source)
		return spec, err
	}
}
