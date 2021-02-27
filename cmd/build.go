package cmd

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"

	"github.com/docker/docker/api/types"
	builder "github.com/groenlid/docker-builder/cmd/builders"
	"github.com/groenlid/docker-builder/cmd/structs"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds the services",
	Long:  `Builds the services under the current working directory`,
	Run: func(cmd *cobra.Command, args []string) {
		runBuild(cmd, args)
	},
}

var foldersToSkip = []string{"node_modules", ".git", "bin"}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	buildCmd.Flags().StringP("registryUsername", "u", "", "The username for the docker registry being used")
	buildCmd.Flags().StringP("registryPassword", "p", "", "The password for the docker registry being used")
	buildCmd.Flags().StringP("registry", "r", "", "The docker registry being used")

	buildCmd.MarkFlagRequired("registryUsername")
	buildCmd.MarkFlagRequired("registryPassord")

}

func runBuild(cmd *cobra.Command, args []string) {
	digestCachePath := ".digestcache"
	digestCache := getDigestCache(digestCachePath)

	flags := cmd.Flags()
	dockerusername, _ := flags.GetString("registryUsername")
	dockerpassword, _ := flags.GetString("registryPassword")
	dockerregistry, _ := flags.GetString("registry")

	authString, err := getRegistryAuthString(dockerusername, dockerpassword, dockerregistry)
	if err != nil {
		log.Fatalln(err)
	}
	configurations, err := findYT3ConfigurationFiles(".")

	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()

	buildAndPushImages(ctx, configurations, authString)

	persitDigestCache(digestCachePath, digestCache)
}

func cleanArtifactFolders() {

}

type digestcache map[string]string

func getDigestCache(path string) digestcache {
	file, err := os.Open(path)
	cache := make(digestcache)
	if err != nil {
		log.Println(err)
		return cache
	}

	log.Printf("Found the digestcache file")

	defer file.Close()

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		return cache
	}

	deserializeError := json.Unmarshal(fileContent, &cache)

	if deserializeError != nil {
		log.Println(deserializeError)
	}
	return cache
}

func persitDigestCache(path string, digestToSave digestcache) {

	bytes, err := json.Marshal(digestToSave)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(bytes))
	writeErr := ioutil.WriteFile(path, bytes, 0666)
	if writeErr != nil {
		log.Println(writeErr)
	}
}

func getRegistryAuthString(username string, password string, dockerregistry string) (string, error) {
	authConfig := types.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: dockerregistry,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

func findYT3ConfigurationFiles(sourceDirectory string) ([]structs.ConfigurationWithProjectPath, error) {
	configName := "ytbdsettings.json"
	var configs []structs.ConfigurationWithProjectPath

	err := filepath.Walk(sourceDirectory, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		if info.IsDir() {
			for _, folderToSkip := range foldersToSkip {
				if folderToSkip == info.Name() {
					return filepath.SkipDir
				}
			}
		}

		if !info.Mode().IsRegular() || info.Name() != configName {
			return nil
		}

		log.Printf("Found configuration file at path: %v", path)

		fileContent, err := ioutil.ReadFile(path)
		if err != nil {
			log.Println(err)
			return nil
		}

		configuration := structs.Configuration{}
		deserializeError := json.Unmarshal(fileContent, &configuration)

		if deserializeError != nil {
			log.Println(deserializeError)
			return nil
		}

		configs = append(configs, structs.ConfigurationWithProjectPath{
			Configuration: configuration,
			ProjectPath:   filepath.Dir(path),
		})
		return nil
	})

	return configs, err
}

func buildAndPushImages(ctx context.Context, configurations []structs.ConfigurationWithProjectPath, auth string) {
	for _, configuration := range configurations {
		buildDockerImage(ctx, configuration)
	}
}

func getContextDirForConfiguration(ctx context.Context, configuration structs.ConfigurationWithProjectPath) (*tar.Reader, error) {
	_, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	file := filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar", configuration.ServiceName))

	log.Printf("Creating tar fil at path %s from path %s", file, configuration.ProjectPath)

	tarError := tarDirectory(configuration.ProjectPath, file)

	if tarError != nil {
		return nil, tarError
	}

	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(reader)
	return tarReader, nil
}

func buildDockerImage(ctx context.Context, configuration structs.ConfigurationWithProjectPath) {
	os.Setenv("DOCKER_BUILDKIT", "1")
	os.Setenv("BUILDKIT_PROGRESS", "plain")
	log.Printf("Building project %s", configuration.ServiceName)

	arguments, err := builder.Manager.GetBuildArgumentsForProject(configuration)
	if err != nil {
		log.Fatalln(err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln(err)
	}

	tarReader, err := getContextDirForConfiguration(ctx, configuration)

	if err != nil {
		log.Fatalln(err)
	}
	log.Println(configuration.ServiceName, arguments)
	log.Print(cli, tarReader)
	/*
		buildOptions := types.ImageBuildOptions{
			Dockerfile: arguments.Dockerfile,
		}

		cli.ImageBuild(ctx, tarReader, buildOptions)
	*/
}

func tarDirectory(source, target string) error {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				for _, folderToSkip := range foldersToSkip {
					if folderToSkip == info.Name() {
						return filepath.SkipDir
					}
				}
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}

func pushImage() {

}

func copyDeloymentArtifactsToOutputFolder() {

}
