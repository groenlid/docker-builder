package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var NodeBuilder = &Builder{
	BuilderName: "nodejs",
	CanBuildProject: func(conf structs.ConfigurationWithProjectPath) bool {
		return conf.Builder.Type == "nodejs"
	},
	GetBuildArguments: func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
		return nil, nil
	},
}
