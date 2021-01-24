package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var NodeBuilder = &Builder{
	BuilderName: "Nodejs",
	CanBuildProject: func(conf structs.ConfigurationWithProjectPath) bool {
		return false
	},
}
