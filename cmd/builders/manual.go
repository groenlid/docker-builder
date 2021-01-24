package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var ManualBuilder = &Builder{
	BuilderName: "manual",
	CanBuildProject: func(conf structs.ConfigurationWithProjectPath) bool {
		return conf.Builder.Type == "" || conf.Builder.Type == "manual"
	},
}
