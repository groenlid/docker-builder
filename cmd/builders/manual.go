package builder

import (
	"github.com/groenlid/docker-builder/cmd/structs"
)

var ManualBuilder = &Builder{
	BuilderNames: []string {"manual", ""},
	GetBuildArguments: func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
		return nil, nil
	},
}
