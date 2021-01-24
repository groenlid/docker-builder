package builder

import (
	"errors"
	"log"

	"github.com/groenlid/docker-builder/cmd/structs"
)

type Builder struct {
	BuilderName                  string
	CanBuildProject              func(conf structs.ConfigurationWithProjectPath) bool
	GetDockerfileContentForBuild func(conf structs.ConfigurationWithProjectPath) (string, error)
}

type BuilderManager struct {
	Builders []*Builder
}

func (m *BuilderManager) PrintBuilders() {
	for _, builder := range m.Builders {
		log.Println(builder.BuilderName)
	}
}

func (m *BuilderManager) GetBuilderForProject(conf structs.ConfigurationWithProjectPath) (*Builder, error) {
	for _, builder := range m.Builders {
		if builder.CanBuildProject(conf) {
			return builder, nil
		}
	}
	return nil, errors.New("No builder found for configuration at path :" + conf.ProjectPath)
}

var Manager = &BuilderManager{
	Builders: []*Builder{
		NodeBuilder,
		DotnetBuilder,
		ManualBuilder,
	},
}
