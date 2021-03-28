package builder

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/groenlid/docker-builder/cmd/structs"
)

type ManualBuilderConf struct {
	BuildContext string `json:buildcontext`
	DockerFile   string `json:dockerfile`
}

/*
export interface ManualBuilder {
type: 'manual',
buildcontext?: 'root' | 'projectdir';
dockerfile?: string;
}
*/

/*
	dockerfilepath: settings.projectPath + sep + ((builder && builder.dockerfile) || 'Dockerfile'),
    dockerbuildcontextpath: builder?.buildcontext && builder.buildcontext === 'projectdir' ? settings.projectPath : sourcesDirectory
*/

var ManualBuilder = &Builder{
	BuilderNames: []string{"manual", ""},
	GetBuildArguments: func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
		builderConfig := &ManualBuilderConf{}
		err := json.Unmarshal(conf.Builder, builderConfig)
		if err != nil {
			return nil, err
		}

		contextPaths := map[string]string{}

		if builderConfig.BuildContext == "root" || builderConfig.BuildContext == "" {
			return &BuildArguments{
				DockerBuildContextPaths: map[string]string{
					".": "",
				},
				DockerFilePath: filepath.Join(conf.ProjectPath, builderConfig.DockerFile),
			}, nil

		} else if builderConfig.BuildContext == "projectdir" {
			return &BuildArguments{
				DockerBuildContextPaths: map[string]string{
					".": "",
				},
				DockerFilePath: filepath.Join(conf.ProjectPath, builderConfig.DockerFile),
			}, nil
			contextPaths[conf.ProjectPath] = ""
		}

		return nil, fmt.Errorf("invalid buildContext value. given %s", builderConfig.BuildContext)
	},
}
