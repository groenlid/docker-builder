import { readdirasync, readFileasync, replaceAll, writeFileAsync } from "../../lib/helpers";
import { tmpdir, EOL } from "os";
import { sep, normalize, basename, dirname, win32, relative } from 'path';
import { v4 } from "uuid";
import { BuildSettingsWithProjectPath, DotnetBuilder } from "../../lib/config";
import { BuilderBuildArguments } from "../../build";
import { sourcesDirectory } from "../../lib/constants";
import { dockerTemplate } from "./templates/Dockerfile";

const findProjectFileInPath = async (path: string): Promise<string | undefined> => {
    const files = await readdirasync(path);
    const projectFile = files.find(file => file.name.includes(".csproj"));
    return projectFile ? projectFile.name : undefined;
}

const mergeSets = (...args: Array<string[] | Set<string>>): Set<string> => {
    const newSet = new Set<string>();
    for(const set of args)
        set.forEach((value: string) => newSet.add(value))
    return newSet;
}

const findDependenciesInProjectFile = async (projectFolder: string, projectFile: string): Promise<string[]> => {
    let dependencies = new Set<string>([projectFolder + sep + projectFile]);
    const content = await readFileasync(projectFolder + sep + projectFile);
    const references = (content.match(/ProjectReference\ Include\=\"(.*)\"/g) || [])
        .map(r => replaceAll(r, 'ProjectReference Include="', ""))
        .map(r => replaceAll(r, '"', ""));
    
    for(const reference of references) {
        const dependencyProjectPath = normalize(projectFolder + sep + replaceAll(reference, win32.sep, sep));
        const foldername = dirname(dependencyProjectPath);
        const projectfilename = basename(dependencyProjectPath); 
        const dependenciesInsideReference = await findDependenciesInProjectFile(foldername, projectfilename);
        dependencies = mergeSets(dependencies, dependenciesInsideReference);
    }

    return Array.from(dependencies);
}

const findPackageDependenciesInProjectFile = async (projectFilesPaths: string[]): Promise<string[]> => {
    const packageDeps = new Set<string>();
    for(const projectFilePath of projectFilesPaths) {
        const content = await readFileasync(projectFilePath);
        // <PackageReference Include="Microsoft.AspNetCore.App" />
        // <PackageReference Include="Microsoft.AspNetCore.Hosting" Version="2.2.7" />
        const references = (content.match(/PackageReference\ Include\=\"(.*)\"/g) || [])
            .map(r => replaceAll(r, 'PackageReference Include="', ""))
            .map(r => replaceAll(r, '"', ""))
            .map(r => r.split(" ")[0]);

        for(const reference of references) {
            packageDeps.add(reference);
        }
    }

    return Array.from(packageDeps);
}

const DOCKER_SRC = "/src/";

const getDockerCopyCommand = (from: string, to: string) => {
    return `COPY ${from} ${DOCKER_SRC}${to}`;
}

const getDockerCopyCommandForDependencies = (dependencies: string[]) => dependencies.map(dep => getDockerCopyCommand(dep, dep)).join(EOL);

interface IReplaceValues {
    COPY_PROJECT_DEPENDENCIES: string;
    COPY_PROJECT_DEPENDENCIES_PROJECTFILES: string;
    COPY_PROJECTFILE: string;
    PROJECTDIR: string;
    COPY_PROJECTDIR: string;
    PROJECTNAME: string;
    DOCKER_RUNTIME_IMAGE: string;
}

const getDockerFileContent = async (replaceValues: IReplaceValues) => {
    let content = dockerTemplate;
    for(const entry of Object.entries(replaceValues)){
        content = replaceAll(content, `#{${entry[0]}}#`, entry[1]);
    }
    return content;
}

const getRuntime = (builder: DotnetBuilder, packageDependencies: string[]): string => {
    
    const runtimeImage = {
        'runtime': 'mcr.microsoft.com/dotnet/core/runtime',
        'aspnet': 'mcr.microsoft.com/dotnet/core/aspnet'
    };

    if(builder.dotnetruntime)
        return runtimeImage[builder.dotnetruntime];

    const aspnetDependencies = ['Microsoft.AspNetCore.App', 'Microsoft.AspNetCore.All'];
    if(aspnetDependencies.some(pkg => packageDependencies.includes(pkg)))
        return runtimeImage.aspnet;
    
    return runtimeImage.runtime;
};

const getBuildArguments = async (setting: BuildSettingsWithProjectPath, builder: DotnetBuilder): Promise<BuilderBuildArguments> => {
    const dir = tmpdir();
    const filename = v4();
    const projectFilePath = await findProjectFileInPath(setting.projectPath);
    if(!projectFilePath)
        throw new Error(`No dotnet projectfile found in path ${setting.projectPath}`);

    const dockerbuildcontextpath = sourcesDirectory;

    const allProjectPaths = (await findDependenciesInProjectFile(setting.projectPath, projectFilePath)).map(dep => relative(dockerbuildcontextpath, dep));
    const allProjectPackageDependencies = await findPackageDependenciesInProjectFile(allProjectPaths);
    const [ownProjectPath, ...dependencies] = allProjectPaths; 
    const COPY_PROJECT_DEPENDENCIES_PROJECTFILES = getDockerCopyCommandForDependencies(dependencies);
    const COPY_PROJECT_DEPENDENCIES = getDockerCopyCommandForDependencies(dependencies.map(dirname));
    const PROJECTNAME = projectFilePath.replace('.csproj', '');
    const COPY_PROJECTFILE = getDockerCopyCommand(ownProjectPath, ownProjectPath);
    const PROJECTDIR = DOCKER_SRC + relative(dockerbuildcontextpath, setting.projectPath);
    const COPY_PROJECTDIR = getDockerCopyCommand(dirname(ownProjectPath), dirname(ownProjectPath));
    const DOCKER_RUNTIME_IMAGE = getRuntime(builder, allProjectPackageDependencies);
    
    const dockerContent = await getDockerFileContent({ 
        COPY_PROJECTDIR,
        COPY_PROJECTFILE,
        COPY_PROJECT_DEPENDENCIES_PROJECTFILES,
        COPY_PROJECT_DEPENDENCIES,
        PROJECTDIR,
        PROJECTNAME,
        DOCKER_RUNTIME_IMAGE
    });

    const tmpdockerfilepath = dir + sep + filename;
    
    await writeFileAsync(tmpdockerfilepath, dockerContent);
    return {
        dockerbuildcontextpath,
        dockerfilepath: tmpdockerfilepath
    }
}

export default {
    getBuildArguments
}