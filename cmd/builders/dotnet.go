package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var DotnetBuilder = &Builder{
	BuilderNames: []string{"dotnet"},
	GetBuildArguments: func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
		return nil, nil
	},
}
