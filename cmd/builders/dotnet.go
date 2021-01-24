package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var DotnetBuilder = &Builder{
	BuilderName: "Dotnet",
	CanBuildProject: func(conf structs.ConfigurationWithProjectPath) bool {
		return false
	},
}
