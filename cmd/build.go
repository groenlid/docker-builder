package cmd

import (
	"archive/tar"
	"bufio"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

var foldersToSkip = []string{"node_modules", ".git", "bin", ".builder"}
var tmpFolder = ".builder"

func init() {
	rootCmd.AddCommand(buildCmd)
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

		// log.Printf("Found configuration file at path: %v", path)

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

func getHexHasForContent(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

func getHexForFolder(folderPath string) string {
	hasher := md5.New()
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
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

		if info.IsDir() {
			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		reader, err := os.Open(path)

		if err != nil {
			return err
		}
		io.Copy(hasher, reader)
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func getContextFilePath(ctx context.Context, buildArguments *builder.BuildArguments, builderTmpFolder string) (string, error) {

	contextPaths := make([]string, 0, len(buildArguments.DockerBuildContextPaths))
	for k := range buildArguments.DockerBuildContextPaths {
		contextPaths = append(contextPaths, k)
	}
	sort.Strings(contextPaths)

	hashes := []string{}

	for _, item := range contextPaths {
		log.Printf("Fetching hash for folder %s", item)
		start := time.Now()

		hash := getHexForFolder(item)
		hashes = append(hashes, hash)
		elapsed := time.Now().Sub(start)
		log.Printf("Hash for folder %s is %s. It took %s", item, hash, elapsed)
	}

	file := filepath.Join(builderTmpFolder, "contexts", strings.Join(hashes, "-")+".tar")
	return file, nil
}

func createOrReadDockerContext(ctx context.Context, configuration structs.ConfigurationWithProjectPath, buildArguments *builder.BuildArguments, builderTmpFolder string) (*os.File, error) {

	contextPath, err := getContextFilePath(ctx, buildArguments, builderTmpFolder)

	log.Printf("Context path is %s", contextPath)
	if err != nil {
		return nil, err
	}

	reader, err := os.Open(contextPath)

	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		log.Printf("Creating tar file at path %s", contextPath)
		// TODO: Create a tmp file, then rename instead...
		tarError := tarDirectories(buildArguments.DockerBuildContextPaths, contextPath)

		if tarError != nil {
			return nil, tarError
		}

		reader, err := os.Open(contextPath)
		return reader, err
	}
	return reader, nil
}

func buildDockerImage(ctx context.Context, configuration structs.ConfigurationWithProjectPath) {
	os.Setenv("DOCKER_BUILDKIT", "1")
	os.Setenv("BUILDKIT_PROGRESS", "plain")
	log.Printf("Building project %s", configuration.ServiceName)

	buildFolderForProject := path.Join(tmpFolder, configuration.ServiceName)
	mkdirError := os.MkdirAll(buildFolderForProject, 0755)

	if mkdirError != nil {
		log.Fatalln(mkdirError)
	}

	mkcontextdirError := os.MkdirAll(path.Join(buildFolderForProject, "contexts"), 0755)
	if mkcontextdirError != nil {
		log.Fatalln(mkcontextdirError)
	}

	arguments, err := builder.Manager.GetBuildArgumentsForProject(configuration)
	if err != nil {
		log.Fatalln(err)
	}

	if arguments == nil {
		return
	}

	reader, err := createOrReadDockerContext(ctx, configuration, arguments, buildFolderForProject)
	defer reader.Close()

	if err != nil {
		log.Fatalln(err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln(err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags: []string{configuration.ServiceName},
	}

	if arguments.DockerFilePath != "" {
		buildOptions.Dockerfile = arguments.DockerFilePath
	}

	imageBuildResponse, err := cli.ImageBuild(ctx, reader, buildOptions)

	if err != nil {
		log.Fatalln(err)
	}

	id, err := handleDockerBuildResponse(imageBuildResponse.Body)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Id of dockerimage: %s", id)

}

func addFileinfoToTarArchive(tarball *tar.Writer, filePath string, info os.FileInfo, pathInTar string) error {
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}
	header.Name = pathInTar
	if err := tarball.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(tarball, file)
	return err
}

type dockerMessage struct {
	ID          string `json:"id"`
	Stream      string `json:"stream"`
	Error       string `json:"error"`
	ErrorDetail struct {
		Message string
	}
	Status   string `json:"status"`
	Progress string `json:"progress"`
	Aux      struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

func handleDockerBuildResponse(resp io.ReadCloser) (string, error) {
	defer resp.Close()

	scanner := bufio.NewScanner(resp)
	msg := dockerMessage{}
	id := ""
	for scanner.Scan() {
		line := scanner.Bytes()
		msg.ID = ""
		msg.Stream = ""
		msg.Error = ""
		msg.ErrorDetail.Message = ""
		msg.Aux.ID = ""
		msg.Status = ""
		msg.Progress = ""
		if err := json.Unmarshal(line, &msg); err == nil {
			if msg.Error != "" {
				return id, fmt.Errorf("%s", msg.Error)
			}
			if msg.Aux.ID != "" {
				id = msg.Aux.ID
			} else if msg.Status != "" {
				if msg.Progress != "" {
					log.Printf("  %s :: %s :: %s\n", msg.Status, msg.ID, msg.Progress)
				} else {
					log.Printf("  %s :: %s\n", msg.Status, msg.ID)
				}
			} else if msg.Stream != "" {
				log.Printf("  %s", msg.Stream)
			} else {
				log.Printf("Unable to handle line: %s", string(line))
			}
		} else {
			log.Printf("Unable to unmarshal line [%s] ==> %v", string(line), err)
		}
	}

	return id, nil
}

func addPathToTarArchive(tarball *tar.Writer, filePath string, pathInTar string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	addFileinfoToTarArchive(tarball, filePath, stat, pathInTar)
	return nil
}

func tarDirectories(sources map[string]string, target string) error {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	for source, inContext := range sources {

		stat, err := os.Stat(source)

		if !stat.IsDir() {
			addFileinfoToTarArchive(tarball, source, stat, inContext)
			continue
		}
		if err != nil {
			return err
		}

		err = filepath.Walk(source,
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

				if info.IsDir() {
					return nil
				}

				if !info.Mode().IsRegular() {
					return nil
				}

				filepathInTar := strings.TrimPrefix(path, source)
				return addFileinfoToTarArchive(tarball, path, info, filepathInTar)
			})
		if err != nil {
			return err
		}
	}
	return nil

}

func pushImage() {

}

func copyDeloymentArtifactsToOutputFolder() {

}
