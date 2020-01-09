package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	// "github.com/x0rzkov/arXiv2git/golang/pkg/dockerfile"
	// "github.com/codeskyblue/dockerignore"
	// "github.com/docker-library/go-dockerlibrary/manifest"
	// "github.com/novln/docker-parser"
	// "github.com/ansjin/docker-compose-file-parse"
	// "github.com/k0kubun/pp"
	"github.com/x0rzkov/arXiv2git/golang/pkg/dockerfile"
	// "github.com/x0rzkov/dockerfile-json/pkg/dockerfile" // interesting to json
	"github.com/yalp/jsonpath"
)

var dockerParserConfig struct {
	Quiet          bool
	Expand         bool
	JSONPathString string
	JSONPath       jsonpath.FilterFunc
	JSONPathRaw    bool
	BuildArgs      AssignmentsMap
	NonzeroExit    bool
}

func dockerfileParser(path string) ([]byte, error) {
	dockerfile, err := dockerfile.Parse(path)
	if err != nil {
		return nil, err
	}
	if dockerParserConfig.Expand {
		env := buildArgEnvExpander()
		//for _, dockerfile := range dockerfiles {
		dockerfile.Expand(env)
		//}
	}
	// pp.Println(dockerfile)
	return json.MarshalIndent(dockerfile, "", "  ")
}

func buildArgEnvExpander() dockerfile.SingleWordExpander {
	env := make(map[string]string, len(dockerParserConfig.BuildArgs.Values))
	for key, value := range dockerParserConfig.BuildArgs.Values {
		if value != nil {
			env[key] = *value
			continue
		}
		if value, ok := os.LookupEnv(key); ok {
			env[key] = value
		}
	}
	return func(word string) (string, error) {
		if value, ok := env[word]; ok {
			return value, nil
		}
		return "", fmt.Errorf("not defined: $%s", word)
	}
}

// AssignmentsMap is a `flag.Value` for `KEY=VALUE` arguments.
type AssignmentsMap struct {
	Values map[string]*string
	Texts  []string
}

// Help returns a string suitable for inclusion in a flag help message.
func (fv *AssignmentsMap) Help() string {
	separator := "="
	return fmt.Sprintf("a key/value pair KEY[%sVALUE]", separator)
}

// Set is flag.Value.Set
func (fv *AssignmentsMap) Set(v string) error {
	separator := "="
	fv.Texts = append(fv.Texts, v)
	if fv.Values == nil {
		fv.Values = make(map[string]*string)
	}
	i := strings.Index(v, separator)
	if i < 0 {
		fv.Values[v] = nil
		return nil
	}
	value := v[i+len(separator):]
	fv.Values[v[:i]] = &value
	return nil
}

func (fv *AssignmentsMap) String() string {
	return strings.Join(fv.Texts, ", ")
}
