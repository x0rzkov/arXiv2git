package dockerfile

import (
	"github.com/x0rzkov/buildkit/frontend/dockerfile/instructions"
)

type Dockerfile struct {
	MetaArgs []*MetaArg
	Stages   []*Stage
}

type MetaArg struct {
	instructions.ArgCommand `json:"-" yaml:"-"`
	Key                     string  `yaml:"-"`
	DefaultValue            *string `json:"," yaml:","`
	ProvidedValue           *string `json:"," yaml:","`
	Value                   *string `json:"," yaml:","`
}

type From struct {
	Stage   *FromStage `json:",omitempty" yaml:",omitempty"`
	Scratch bool       `json:",omitempty" yaml:",omitempty"`
	Image   *string    `json:",omitempty" yaml:",omitempty"`
}

type FromStage struct {
	Named *string `json:",omitempty" yaml:",omitempty"`
	Index int
}

type Command struct {
	instructions.Command
	Name string
}

type Stage struct {
	instructions.Stage
	Name     *string `json:"As,omitempty" yaml:"as,omitempty"`
	From     From
	Commands []*Command
}
