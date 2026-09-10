package w

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	textTemplate "text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// TemplateHandler is the common execution surface shared by text and HTML templates.
type TemplateHandler interface {
	// Name returns the name of the template.
	Name() string
	// ExecuteTemplate applies the named template to data and writes the output to wr.
	ExecuteTemplate(wr io.Writer, name string, data any) error
	// Execute applies the template to data and writes the output to wr.
	Execute(wr io.Writer, data any) error
	// DefinedTemplates returns a string listing the defined template names.
	DefinedTemplates() string
}

// Template holds a parsed text or HTML template together with its raw source.
type Template struct {
	raw      string
	htmlTmpl *htmlTemplate.Template
	textTmpl *textTemplate.Template
}

// MarshalYAML implements yaml.Marshaler by returning the raw template source.
func (t Template) MarshalYAML() (interface{}, error) {
	return t.raw, nil
}

func (t *Template) parseTextTemplate(raw string) (err error) {
	t.htmlTmpl = nil
	t.textTmpl = textTemplate.New("root").Funcs(funcMaps)
	t.textTmpl, err = t.textTmpl.Parse(raw)
	return err
}

func (t *Template) parseTextFileTemplate(filename string) (err error) {
	t.htmlTmpl = nil
	t.textTmpl = textTemplate.New(filepath.Base(filename)).Funcs(funcMaps)
	t.textTmpl, err = t.textTmpl.ParseFiles(filename)
	return err
}

func (t *Template) parseHTMLFileTemplate(filename string) (err error) {
	t.textTmpl = nil
	t.htmlTmpl = htmlTemplate.New(filepath.Base(filename)).Funcs(funcMaps)
	t.htmlTmpl, err = t.htmlTmpl.ParseFiles(filename)
	return err
}

func (t *Template) parseHTMLTemplate(raw string) (err error) {
	t.textTmpl = nil
	t.htmlTmpl = htmlTemplate.New("root").Funcs(funcMaps)
	t.htmlTmpl, err = t.htmlTmpl.Parse(raw)
	return err
}

// UnmarshalYAML implements yaml.Unmarshaler. It parses a text or HTML template from a YAML scalar,
// using tags such as !text, !html, !file, and !htmlFile.
func (t *Template) UnmarshalYAML(value *yaml.Node) (err error) {
	t.raw = value.Value
	switch value.Tag {
	case "!!str":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseTextTemplate(value.Value)
		default:
			return fmt.Errorf("unknown type: %s", value.Tag)
		}
	case "!text":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseTextTemplate(value.Value)
		default:
			return fmt.Errorf("unknown type: %s", value.Tag)
		}
	case "!file":
		err = t.parseTextFileTemplate(value.Value)
	case "!htmlFile":
		err = t.parseHTMLFileTemplate(value.Value)
	case "!html":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseHTMLTemplate(value.Value)
		default:
			return fmt.Errorf("unknown type: %s", value.Tag)
		}
	default:
		return fmt.Errorf("unknown type: %s", value.Tag)
	}

	return err
}

// MarshalJSON implements json.Marshaler by returning the raw template source.
func (t Template) MarshalJSON() ([]byte, error) {
	return []byte(t.raw), nil
}

var defaultFuncs = textTemplate.FuncMap{
	"toUpper": strings.ToUpper,
	"toLower": strings.ToLower,
	"add":     func(a, b int) int { return a + b },
	"sub":     func(a, b int) int { return a - b },
	"title":   cases.Title(language.AmericanEnglish).String,
	// join is equal to strings.Join but inverts the argument order
	// for easier pipelining in templates.
	"join": func(sep string, s []string) string {
		return strings.Join(s, sep)
	},
	"match": regexp.MatchString,
	"safeHtml": func(text string) htmlTemplate.HTML {
		return htmlTemplate.HTML(text)
	},
	"reReplaceAll": func(pattern, repl, text string) string {
		re := regexp.MustCompile(pattern)
		return re.ReplaceAllString(text, repl)
	},
	"stringSlice": func(s ...string) []string {
		return s
	},
	"toMap": func(s string) map[string]string {
		m := make(map[string]string)
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			panic(err)
		} else {
			return m
		}
	},
	"toJson": func(o interface{}) string {
		data, _ := json.Marshal(o)
		return string(data)
	},
	"safeJson": func(o interface{}) string {
		data, _ := json.Marshal(o)
		return strings.Trim(string(data), "\"")
	},
}

var funcMaps = textTemplate.FuncMap{}

// AddFuncMaps registers a named function that templates can call.
func AddFuncMaps(funcName string, f interface{}) {
	funcMaps[funcName] = f
}

func init() {
	for name, f := range defaultFuncs {
		AddFuncMaps(name, f)
	}
}

// NewTextTemplate parses raw as a text/template and returns a Template.
func NewTextTemplate(raw string) (*Template, error) {
	var t Template
	err := t.parseTextTemplate(raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// NewHTMLTemplate parses raw as an html/template and returns a Template.
func NewHTMLTemplate(raw string) (*Template, error) {
	var t Template
	err := t.parseHTMLTemplate(raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UnmarshalJSON implements json.Unmarshaler. A JSON string is parsed as a text template;
// a single-key object such as {"fn::html": "..."} selects the parser.
func (t *Template) UnmarshalJSON(data []byte) (err error) {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		var text string
		if err = json.Unmarshal(data, &text); err != nil {
			return err
		}
		err = t.parseTextTemplate(text)
		return err
	}
	var tmplFuncs map[string]string
	if err = json.Unmarshal(data, &tmplFuncs); err != nil {
		return err
	}
	if len(tmplFuncs) != 1 {
		return fmt.Errorf("invalid format")
	}
	for name, val := range tmplFuncs {
		switch name {
		case "fn::text":
			err = t.parseTextTemplate(val)
		case "fn::file":
			err = t.parseTextFileTemplate(val)
		case "fn::html":
			err = t.parseHTMLTemplate(val)
		case "fn::htmlFile":
			err = t.parseHTMLFileTemplate(val)
		default:
			return fmt.Errorf("unknown function: %s", name)
		}
		break
	}
	return err
}

// Execute applies the template to data and writes the output to wr.
func (t *Template) Execute(wr io.Writer, data any) error {
	return t.handler().Execute(wr, data)
}

// ExecuteToString applies the template to data and returns the output as a string.
func (t *Template) ExecuteToString(data any) (string, error) {
	var buf bytes.Buffer
	err := t.handler().Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DefinedTemplates returns a string listing the defined template names.
func (t *Template) DefinedTemplates() string {
	return t.handler().DefinedTemplates()
}

// Name returns the name of the underlying template.
func (t *Template) Name() string {
	return t.handler().Name()
}

// ExecuteTemplate applies the named template to data and writes the output to wr.
func (t *Template) ExecuteTemplate(wr io.Writer, name string, data any) error {
	return t.handler().ExecuteTemplate(wr, name, data)
}

func (t *Template) handler() TemplateHandler {
	if t.textTmpl != nil {
		return t.textTmpl
	}
	return t.htmlTmpl
}
