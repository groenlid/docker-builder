package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var DotnetBuilder = &Builder{
	BuilderName: "dotnet",
	CanBuildProject: func(conf structs.ConfigurationWithProjectPath) bool {
		return conf.Builder.Type == "dotnet"
	},
}