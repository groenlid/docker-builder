package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/groenlid/docker-builder/cmd/structs"
)

type Builder struct {
	BuilderNames      []string
	GetBuildArguments func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error)
}

type BuildArguments struct {
	DockerfileContent       string
	DockerBuildContextPaths map[string]string
}

type BuilderManager struct {
	Builders []*Builder
}

func (m *BuilderManager) PrintBuilders() {
	for _, builder := range m.Builders {
		log.Println(builder.BuilderNames)
	}
}

func (m *BuilderManager) GetBuildArgumentsForProject(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
	baseBuilder := &structs.BaseBuilder{}

	err := json.Unmarshal(conf.Builder, &baseBuilder)
	if err != nil {
		return nil, err
	}

	for _, builder := range m.Builders {
		for _, builderName := range builder.BuilderNames {
			if builderName == baseBuilder.Type {
				return builder.GetBuildArguments(conf)
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("No builder found for service %s at path %s", conf.ServiceName, conf.ProjectPath))
}

var Manager = &BuilderManager{
	Builders: []*Builder{
		NodeBuilder,
		DotnetBuilder,
		ManualBuilder,
	},
}
