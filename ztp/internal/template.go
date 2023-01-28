/*
Copyright 2023 Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in
compliance with the License. You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is
distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing permissions and limitations under the
License.
*/

package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	tmpl "text/template"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"
)

// TemplateBuilder contains the data and logic needed to create templates. Don't create objects of
// this type directly, use the NewTemplate function instead.
type TemplateBuilder struct {
	logger logr.Logger
	fsys   fs.FS
	dir    string
}

// Template is a template based on template.Template with some additional functions. Don't create
// objects of this type directly, use the NewTemplate function instead.
type Template struct {
	logger   logr.Logger
	names    []string
	template *tmpl.Template
}

// NewTemplate creates a builder that can the be used to create a template.
func NewTemplate() *TemplateBuilder {
	return &TemplateBuilder{}
}

// SetLogger sets the logger that the template will use to write messages to the log. This is
// mandatory.
func (b *TemplateBuilder) SetLogger(value logr.Logger) *TemplateBuilder {
	b.logger = value
	return b
}

// SetFS sets the filesystem that will be used to read the templates. This is mandatory.
func (b *TemplateBuilder) SetFS(value fs.FS) *TemplateBuilder {
	b.fsys = value
	return b
}

// SetDir loads the templates only from the given directory. This is optional.
func (b *TemplateBuilder) SetDir(value string) *TemplateBuilder {
	b.dir = value
	return b
}

// Build uses the configuration stored in the builder to create a new template.
func (b *TemplateBuilder) Build() (result *Template, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}
	if b.fsys == nil {
		err = errors.New("filesystem is mandatory")
		return
	}

	// Calculate the root directory:
	fsys := b.fsys
	if b.dir != "" {
		fsys, err = fs.Sub(b.fsys, b.dir)
		if err != nil {
			return
		}
	}

	// We need to create the object early because the some of the functions need the pointer:
	t := &Template{
		logger:   b.logger,
		template: tmpl.New(""),
	}

	// Register the functions:
	t.template.Funcs(tmpl.FuncMap{
		"base64":  t.base64Func,
		"execute": t.executeFunc,
		"json":    t.jsonFunc,
	})

	// Find and parse the template files:
	err = t.findFiles(fsys)
	if err != nil {
		return
	}
	err = t.parseFiles(fsys)
	if err != nil {
		return
	}

	// Return the object:
	result = t
	return
}

func (t *Template) findFiles(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		t.names = append(t.names, name)
		return nil
	})
}

func (t *Template) parseFiles(fsys fs.FS) error {
	for _, name := range t.names {
		err := t.parseFile(fsys, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) parseFile(fsys fs.FS, name string) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	text := string(data)
	_, err = t.template.New(name).Parse(text)
	if err != nil {
		return err
	}
	detail := t.logger.V(2)
	if detail.Enabled() {
		detail.Info(
			"Parsed template",
			"name", name,
			"text", text,
		)
	}
	return nil
}

// Execute executes the template with the given name and passing the given input data. It writes the
// result to the given writer.
func (t *Template) Execute(writer io.Writer, name string, data any) error {
	buffer := &bytes.Buffer{}
	err := t.template.ExecuteTemplate(buffer, name, data)
	if err != nil {
		return err
	}
	_, err = buffer.WriteTo(writer)
	if err != nil {
		return err
	}
	detail := t.logger.V(2)
	if detail.Enabled() {
		detail.Info(
			"Executed template",
			"name", name,
			"data", data,
			"text", buffer.String(),
		)
	}
	return nil
}

// Names returns the names of the templates.
func (t *Template) Names() []string {
	return slices.Clone(t.names)
}

// base64Func is a template function that encodes the given data using Base64 and returns the result
// as a string. If the data is an array of bytes it will be encoded directly. If the data is a
// string it will be converted to an array of bytes using the UTF-8 encoding. If the data implements
// the fmt.Stringer interface it will be converted to a string using the String method, and then to
// an array of bytes using the UTF-8 encoding. Any other kind of data will result in an error.
func (t *Template) base64Func(value any) (result string, err error) {
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	case fmt.Stringer:
		data = []byte(typed.String())
	default:
		err = fmt.Errorf(
			"don't know how to encode value of type %T",
			value,
		)
		if err != nil {
			return
		}
	}
	result = base64.StdEncoding.EncodeToString(data)
	return
}

// executeFunc is a template function similar to template.ExecuteTemplate but it returns the result
// instead of writing it to the output. That is useful when some processing is needed after that,
// for example, to encode the result using Base64:
//
//	{{ execute "my.tmpl" . | base64 }}
func (t *Template) executeFunc(name string, data any) (result string, err error) {
	buffer := &bytes.Buffer{}
	executed := t.template.Lookup(name)
	err = executed.Execute(buffer, data)
	if err != nil {
		return
	}
	result = buffer.String()
	return
}

// jsonFunc is a template function that encodes the given data as JSON. This can be used, for
// example, to encode as a JSON string the result of executing other function. For example, to
// create a JSON document with a 'content' field that contains the text of the 'my.tmpl' template:
//
//	"content": {{ execute "my.tmpl" . | json }}
//
// Note how that the value of that 'content' field doesn't need to sorrounded by quotes, because the
// 'json' function will generate a valid JSON string, including those quotes.
func (t *Template) jsonFunc(data any) (result string, err error) {
	text, err := json.Marshal(data)
	if err != nil {
		return
	}
	result = string(text)
	return
}