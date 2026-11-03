package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
)

// YAML can name mounted secret files, but direct credentials belong only in
// explicit environment variables. Check each layer before a later override can
// conceal the original inline secret. The parser never formats secret values.
type filesYAML struct{ koanf.Parser }

func filesYAMLParser() filesYAML { return filesYAML{Parser: yaml.Parser()} }

func (p filesYAML) Unmarshal(data []byte) (map[string]any, error) {
	values, err := p.Parser.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	if err := rejectInlineS3Secrets(values, ""); err != nil {
		return nil, err
	}
	return values, nil
}

func rejectInlineS3Secrets(values map[string]any, prefix string) error {
	for key, value := range values {
		path := prefix + strings.ToLower(key)
		switch path {
		case "files.s3.access_key_id", "files.s3.secret_access_key", "files.s3.session_token":
			if value != nil && value != "" {
				return fmt.Errorf("%s: direct secrets must use an environment variable, or use a _file source", path)
			}
		}
		if nested, ok := value.(map[string]any); ok {
			if err := rejectInlineS3Secrets(nested, path+"."); err != nil {
				return err
			}
		}
	}
	return nil
}
