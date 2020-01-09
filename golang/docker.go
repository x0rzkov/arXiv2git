package main

import (
	// "github.com/x0rzkov/arXiv2git/golang/pkg/dockerfile"
	// "github.com/codeskyblue/dockerignore"
	// "github.com/docker-library/go-dockerlibrary/manifest"
	// "github.com/novln/docker-parser"
	// "github.com/ansjin/docker-compose-file-parse"
	"github.com/x0rzkov/dockerfile-json/pkg/dockerfile" // interesting to json
	// "github.com/yalp/jsonpath"
)

func dockerfileParser(paths ...string) ([]*dockerfile.Dockerfile, error) {
	var dockerfiles []*dockerfile.Dockerfile
	for _, path := range paths {
		dockerfile, err := dockerfile.Parse(path)
		if err != nil {
			return nil, err
		}
		dockerfiles = append(dockerfiles, dockerfile)
	}
	return dockerfiles, nil
}
