package builder

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/groenlid/docker-builder/cmd/structs"
)

type DotnetBuilderConfig struct {
	Type          string `json:"type"`
	DotnetRuntime string `json:"dotnetruntime"`
	//dotnetruntime?: 'runtime' | 'aspnet'
}

func findProjectFileInPath(path string) (fs.FileInfo, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if strings.Contains(file.Name(), ".csproj") {
			return file, nil
		}
	}

	return nil, fmt.Errorf("could not find project file in path %s", path)
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func findPackageDependenciesInProjectFile(projectFilesPaths []string) ([]string, error) {
	dependencies := []string{}
	for _, projectFilePath := range projectFilesPaths {
		content, err := os.ReadFile(projectFilePath)
		if err != nil {
			return nil, err
		}
		re := regexp.MustCompile(`PackageReference Include="(.*)"`)

		matches := re.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			dependencies = append(dependencies, match[1])
		}
	}
	return dependencies, nil
}

func findProjectDependenciesInProjectFile(projectfolder string, projectfile string) ([]string, error) {
	projectFilePath := path.Join(projectfolder, projectfile)
	projectFileContent, err := os.ReadFile(projectFilePath)

	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`ProjectReference Include="(.*)"`)

	matches := re.FindAllStringSubmatch(string(projectFileContent), -1)

	dependencies := []string{}

	for _, dependency := range matches {
		paths := strings.Split(dependency[1], `\`)
		dependencyPath := projectfolder
		for _, pathFragment := range paths {
			dependencyPath = path.Join(dependencyPath, pathFragment)
		}
		dependencyDir := path.Dir(dependencyPath)
		dependencies = append(dependencies, dependencyPath)
		subdependencies, err := findProjectDependenciesInProjectFile(dependencyDir, strings.Replace(dependencyPath, dependencyDir, "", 1))
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, subdependencies...)
	}
	return unique(dependencies), nil
}

const DOCKER_SRC = "/src/"

func getDockerCopyCommand(from string, to string) string {
	return fmt.Sprintf(`COPY %s %s`, from, path.Join(DOCKER_SRC, to))
}

func getDockerCopyCommandForDependency(from []string) string {
	copyCommands := []string{}
	for _, item := range from {
		copyCommands = append(copyCommands, getDockerCopyCommand(item, item))
	}
	return strings.Join(copyCommands, "\n")
}

func getDirOfPaths(paths []string) []string {
	dirs := []string{}
	for _, item := range paths {
		dirs = append(dirs, path.Dir(item))
	}
	return dirs
}

func getDotnetRuntime(builderConfig *DotnetBuilderConfig, projectfiles []string) (string, error) {
	runtimeImage := map[string]string{
		"runtime": "mcr.microsoft.com/dotnet/core/runtime",
		"aspnet":  "mcr.microsoft.com/dotnet/core/aspnet",
	}

	if builderConfig.DotnetRuntime != "" {
		return runtimeImage[builderConfig.DotnetRuntime], nil
	}

	aspnetDependencies := []string{
		"Microsoft.AspNetCore.App",
		"Microsoft.AspNetCore.All",
	}

	dependencies, err := findPackageDependenciesInProjectFile(projectfiles)
	if err != nil {
		return "", err
	}

	for _, dependency := range dependencies {
		for _, aspnetDedependency := range aspnetDependencies {
			if aspnetDedependency == dependency {
				return runtimeImage["aspnet"], nil
			}
		}
	}

	return runtimeImage["runtime"], nil
}

var DotnetBuilder = &Builder{
	BuilderNames: []string{"dotnet"},
	GetBuildArguments: func(conf structs.ConfigurationWithProjectPath) (*BuildArguments, error) {
		builderConfig := &DotnetBuilderConfig{}
		err := json.Unmarshal(conf.Builder, builderConfig)
		if err != nil {
			return nil, err
		}

		projectFile, err := findProjectFileInPath(conf.ProjectPath)
		if err != nil {
			return nil, err
		}

		log.Printf("Found projectfile in %s", projectFile.Name())

		projectDependencies, err := findProjectDependenciesInProjectFile(conf.ProjectPath, projectFile.Name())

		if err != nil {
			return nil, err
		}

		log.Printf("%q\n", projectDependencies)

		copyProjectDependenciesProjectFiles := getDockerCopyCommandForDependency(projectDependencies)
		projectDir := path.Join(DOCKER_SRC, conf.ProjectPath)
		copyProjectDependencies := getDockerCopyCommandForDependency(getDirOfPaths(projectDependencies))
		projectName := strings.Replace(projectFile.Name(), ".csproj", "", 1)
		dockerRuntimeImage, err := getDotnetRuntime(builderConfig, projectDependencies)

		if err != nil {
			return nil, err
		}

		dockercontent := fmt.Sprintf(`
			 FROM mcr.microsoft.com/dotnet/core/sdk:3.1 AS build-env

			# Copy csproj and restore as distinct layers
			%s
		
			WORKDIR %s
			RUN dotnet restore
		
			# Copy everything else and build
			%s
		
			RUN dotnet publish -c Release -o out
		
			# Build runtime image
			FROM %s:3.1
			WORKDIR /app
			COPY --from=build-env %s/out .
			ENTRYPOINT ["dotnet", "%s.dll"]
		`, copyProjectDependenciesProjectFiles, projectDir, copyProjectDependencies, dockerRuntimeImage, projectDir, projectName)

		log.Println(dockercontent)

		tmpDir, err := ioutil.TempDir("", "")

		if err != nil {
			return nil, err
		}

		dockerFilePath := path.Join(tmpDir, "Dockerfile")
		bytesToSend := []byte(dockercontent)
		err = os.WriteFile(dockerFilePath, bytesToSend, 0755)
		if err != nil {
			return nil, err
		}

		arguments := &BuildArguments{
			DockerBuildContextPaths: map[string]string{
				conf.ProjectPath: "",
				tmpDir:           "",
			},
		}

		return arguments, nil
	},
}
