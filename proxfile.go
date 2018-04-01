package prox

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// DefaultStructuredOutput is the default configuration for processes that do
// not specify structured log output specifically.
var DefaultStructuredOutput = StructuredOutput{
	Format:       "auto",
	MessageField: "msg",
	LevelField:   "level",
	TagColors: map[string]string{
		"error": "red",
		"fatal": "red",
	},
	TaggingRules: []TaggingRule{
		{
			Tag:   "error",
			Field: "level",
			Value: "/(ERR(O|OR)?)|(WARN(ING)?)/i",
		},
		{
			Tag:   "fatal",
			Field: "level",
			Value: "/FATAL?|PANIC/i",
		},
	},
}

type Proxfile struct {
	Processes map[string]ProxfileProcess
}

type ProxfileProcess struct {
	Script string
	Env    []string

	Format string // e.g. json
	Fields struct {
		Message string
		Level   string
	}
	Tags map[string]struct {
		Color     string
		Condition struct {
			Field string
			Value string
		}
	}
}

// proxfileProcess is a 1-1 copy of the ProxfileProcess type to work around infinite recursion when unmarshalling this
// type from YAML. Every field that is added to one field must also be added to the other.
type proxfileProcess struct {
	Script string
	Env    []string

	Format string
	Fields struct {
		Message string
		Level   string
	}
	Tags map[string]struct {
		Color     string
		Condition struct {
			Field string
			Value string
		}
	}
}

// UnmarshalYAML implements the gopkg.in/yaml.v2.Unmarshaler interface.
func (p *ProxfileProcess) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&p.Script)
	if err == nil {
		return nil
	}

	var pp proxfileProcess
	err = unmarshal(&pp)
	if err != nil {
		return err
	}

	*p = ProxfileProcess(pp)
	return nil
}

func ParseProxFile(reader io.Reader, env Environment) ([]Process, error) {
	var proxfile Proxfile
	err := yaml.NewDecoder(reader).Decode(&proxfile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode Proxfile as YAML")
	}

	var processes []Process
	for name, pp := range proxfile.Processes {
		env := NewEnv(env.List())
		env.SetAll(pp.Env)

		p := Process{
			Name:             strings.TrimSpace(name),
			Script:           strings.TrimSpace(pp.Script),
			Env:              env,
			StructuredOutput: DefaultStructuredOutput, // TODO: move this logic so processes defined by a "Procfile" also get these defaults
		}

		if pp.Format != "" {
			p.StructuredOutput.Format = pp.Format
		}

		if pp.Fields.Message != "" {
			p.StructuredOutput.MessageField = pp.Fields.Message
		}

		if pp.Fields.Level != "" {
			p.StructuredOutput.LevelField = pp.Fields.Level
		}

		if p.StructuredOutput.Format == "json" {
			for tag, tagDef := range pp.Tags {
				// Note that the default tags can be overwritten by defining a
				// new tagging action wit the same tag name (i.e. "error" or "fatal").
				p.StructuredOutput.TaggingRules = append(p.StructuredOutput.TaggingRules, TaggingRule{
					Tag:   tag,
					Field: tagDef.Condition.Field,
					Value: tagDef.Condition.Value,
				})
				p.StructuredOutput.TagColors[tag] = tagDef.Color
			}
		}

		processes = append(processes, p)
	}

	return processes, nil
}
