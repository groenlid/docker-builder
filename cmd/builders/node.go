package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/groenlid/docker-builder/cmd/structs"
)

type NodejsBuilderConfig struct {
	Type         string `json:"type"`
	NodeVersion  string `json:"nodeversion"`
	BuildCommand string `json:"buildcommand"`
	RunCommand   string `json:"runcommand"`
}

type NodeProjectType int

const (
	Unknown NodeProjectType = iota
	Npm
	Yarn
)

func GetNodeProjectType(conf structs.ConfigurationWithProjectPath) (NodeProjectType, error) {
	direntries, err := os.ReadDir(conf.ProjectPath)
	if err != nil {
		return Unknown, err
	}

	yarnLockIsFound := false
	packacgeLockIsFound := false
	for _, direntry := range direntries {
		if direntry.Name() == "yarn.lock" {
			yarnLockIsFound = true
		}
		if direntry.Name() == "package-lock.json" {
			packacgeLockIsFound = true
		}
	}

	if yarnLockIsFound && packacgeLockIsFound {
		return Unknown, errors.New(fmt.Sprintf("Both yarn.lock and package-lock.json are found in the project %s at path %s. Please remove one of them", conf.ServiceName, conf.ProjectPath))
	}

	if yarnLockIsFound {
		return Yarn, nil
	}
	if packacgeLockIsFound {
		return Npm, nil
	}
	return Unknown, errors.New(fmt.Sprintf("Neither yarn.lock and package-lock.json were found in the project %s at path %s. Please add one of them.", conf.ServiceName, conf.ProjectPath))
}

func getInstallCommand(projectType NodeProjectType) string {
	switch projectType {
	case Unknown:
		return ""
	case Npm:
		return "npm ci"
	case Yarn:
		return "yarn"
	}
	return ""
}

var NodeBuilder = &Builder{
	BuilderNames: []string{"nodejs"},
	GetBuildArguments: func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
		builderConfig := &NodejsBuilderConfig{}
		err := json.Unmarshal(conf.Builder, builderConfig)
		if err != nil {
			return nil, err
		}

		nodeProjectType, err := GetNodeProjectType(conf)

		if err != nil {
			return nil, err
		}

		if nodeProjectType == Unknown {
			return nil, errors.New(fmt.Sprintf("Could not get node project type for project %s at path %s", conf.ServiceName, conf.ProjectPath))
		}

		lockFile := "yarn.lock"
		if nodeProjectType == Npm {
			lockFile = "package-lock.json"
		}

		installCommand := getInstallCommand(nodeProjectType)

		buildCommand := ""
		if builderConfig.BuildCommand != "" {
			buildCommand = fmt.Sprintf("RUN %s", builderConfig.BuildCommand)
		}

		if builderConfig.RunCommand == "" {
			return nil, errors.New(fmt.Sprintf("No runcommand given in project %s at path %s", conf.ServiceName, conf.ProjectPath))
		}

		dockercontent := fmt.Sprintf(`
			FROM node:%s-alpine

			WORKDIR /usr/src/app
			COPY package.json %s ./

			%s

			COPY / ./

			%s

			CMD %s
		`, builderConfig.NodeVersion, lockFile, installCommand, buildCommand, builderConfig.RunCommand)

		arguments := &BuildArguments{
			DockerfileContent:       dockercontent,
			DockerBuildRootFolder:   conf.ProjectPath,
			DockerBuildContextPaths: []string{conf.ProjectPath},
		}

		log.Println(dockercontent, arguments)
		return arguments, nil
	},
}
