import { cpus } from 'os'
import { artifactfile, containerRegistry, artifactfolder, configurationFile, buildId, excludedFolders, sourcesDirectory, getArgumentValue, dockerLoginUser, dockerLoginPassword } from './lib/constants'
import { sep } from 'path'
import { Aigle } from 'aigle'
import { readFileasync, readdirasync, createFolder, writeFileAsync, spawnProcess, copyFileAsync, replaceTokens } from './lib/helpers'
import { createLogger, isDebugging, ILogger } from './lib/logger'
import del from 'del'
import { performance } from 'perf_hooks';
import { BuildSettings, BuildDecoder, BuildSettingsWithProjectPath, BuildSettingsWithDockerBuildCacheId, BuildSettingsWithDigest } from './lib/config'
import dotnetbuilder from './builders/dotnet';
import manualbuilder from './builders/manual';
import nodejsbuilder from './builders/node';

const globalLogger = createLogger("build-service")

const getDockerImageNameWithTag = (servicename: string) => `${containerRegistry}/${servicename}:${buildId}`

const getNumberOfConcurrentBuilders = (): number => isDebugging() ? 1 : Math.min(cpus().length, 4);

const readConfigurationFile = async (...pathPart: string[]): Promise<BuildSettings | undefined> => {
    const path = pathPart.join(sep);
    const fileContent = await readFileasync(path);
    return BuildDecoder.runWithException(JSON.parse(fileContent));
}

const shouldIncludeConfigurationFile = (logger: ILogger, buildSettings: BuildSettings) => {
    const only = getArgumentValue('only')
    if(!only) return true;
    const shouldInclude = only.includes(buildSettings.servicename);
    if(!shouldInclude)
        logger.debug(`Skipping project ${buildSettings.servicename} because the --only parameter was used`);
    return shouldInclude;
}

export const findConfigurationFiles = async (path: string): Promise<BuildSettingsWithProjectPath[]> => {
    let configurationFilesUnderPath: BuildSettingsWithProjectPath[] = []
    const filesAndFoldersInPath = await readdirasync(path)
    for(const fileOrFolder of filesAndFoldersInPath) {
        if(fileOrFolder.isFile() && fileOrFolder.name.toLocaleLowerCase() === configurationFile) {
            globalLogger.info(`Found configurationfile in ${path}`)
            const configurationFile = await readConfigurationFile(path, fileOrFolder.name)
            if(configurationFile && shouldIncludeConfigurationFile(globalLogger, configurationFile))
                configurationFilesUnderPath.push({ 
                    ...configurationFile,
                    projectPath: path 
                })
            continue;
        }
        if(fileOrFolder.isDirectory() && !excludedFolders.includes(fileOrFolder.name)) {
            const configurationFilesForSubpath = await findConfigurationFiles(path + sep + fileOrFolder.name)
            configurationFilesUnderPath = [...configurationFilesUnderPath, ...configurationFilesForSubpath]
        }
    }
    return configurationFilesUnderPath
}

export interface BuilderBuildArguments {
    dockerfilepath: string;
    dockerbuildcontextpath: string;
}

const getBuildArguments = (settings: BuildSettingsWithProjectPath): Promise<BuilderBuildArguments> => {
    const { builder } = settings
    
    if(!builder || builder.type === 'manual')
        return manualbuilder.getBuildArguments(settings, builder);
    if(builder.type === 'dotnet')
        return dotnetbuilder.getBuildArguments(settings, builder);
    if(builder.type === 'nodejs')
        return nodejsbuilder.getBuildArguments(settings, builder);
    
    throw new Error('Builder not supported.');
}

const runBuild = async (settings: BuildSettingsWithProjectPath): Promise<BuildSettingsWithDockerBuildCacheId> => {
    const logger = createLogger(settings.servicename)
    logger.debug('Running build for ' + settings.servicename + " with settings " + JSON.stringify(settings));
    const before = performance.now();
    const buildArguments = await getBuildArguments(settings);

    logger.debug(`Dockerpath: ${buildArguments.dockerfilepath}, dockerTag: ${getDockerImageNameWithTag(settings.servicename)}, dockerbuildcontext: ${buildArguments.dockerbuildcontextpath}, cwd: ${sourcesDirectory}`);

    const result = await spawnProcess('docker', ['build', '-f', buildArguments.dockerfilepath, '-t', `${getDockerImageNameWithTag(settings.servicename)}`, buildArguments.dockerbuildcontextpath], { env: process.env, cwd: sourcesDirectory }, logger)
    
    const preText = "Successfully built "
    const position = result.lastIndexOf(preText) + preText.length;
    const hashLength = 12;
    const dockerCacheId = result.substr(position, hashLength);
    logger.info(`Building project done. Took: ${((performance.now() - before) / 1000).toFixed(0)} seconds`)
    return { ...settings, dockerCacheId }
}

const buildAndPushImages = async (settings: BuildSettingsWithProjectPath[], digestCache: digestCache): Promise<BuildSettingsWithDigest[]> => {
    const numberOfConcurrentBuilders = getNumberOfConcurrentBuilders();
    return new Promise((resolve, reject) => {
        Aigle.mapValuesLimit(settings, numberOfConcurrentBuilders, buildAndPushImage(digestCache))
            .then(a => resolve(Object.values(a)))
            .catch(err => reject(err))
    });
};

const buildAndPushImage = (digestCache: digestCache) => async (settings: BuildSettingsWithProjectPath): Promise<BuildSettingsWithDigest> => {
    const builtImage = await runBuild(settings);
    return pushBuild(builtImage, digestCache);
}

/* Pushes the build to ACR and returns the digest back */
const pushBuild = async (settings: BuildSettingsWithDockerBuildCacheId, digestCache: digestCache): Promise<BuildSettingsWithDigest> => {
    const logger = createLogger(settings.servicename)
    const acrDigest = digestCache[settings.dockerCacheId];
    if(acrDigest) {
        logger.info(`DockerCacheId was found in digestcache. Returning cached value`);
        return { ...settings, acrDigest };
    }
    logger.info(`Pushing build for project`)
    const before = performance.now();
    const result = await spawnProcess('docker', ['push', getDockerImageNameWithTag(settings.servicename)], { env: process.env, cwd: process.cwd() }, logger)
    logger.info(`Pushed build for project. Took ${((performance.now() - before) / 1000).toFixed(0)} seconds.`)
    const digestStart = result.lastIndexOf("digest:")
    const digestEnd = result.lastIndexOf("size:")

    if(digestStart === -1 && digestEnd === -1)
        throw new Error('Could not find digest from repository for project ' + settings.servicename);
    
    const digest = result.substring(digestStart + "digest:".length, digestEnd).trim();
    logger.info(`Digest is ${digest}`);
    digestCache[settings.dockerCacheId] = digest;
    return {...settings, acrDigest: digest };
}

const digestCachePath = ".digestcache";
type digestCache = Record<string, string | undefined>;
const getDigestCache = async (): Promise<digestCache> => {
    try {
        const cacheString = await readFileasync(digestCachePath);
        return JSON.parse(cacheString) as Record<string, string>;
    } catch(err) {
        globalLogger.info("Could not fetch or deserialize digestcache.");
        return {};
    }
}

const persistDigestCache = (cache: digestCache) => writeFileAsync(digestCachePath, JSON.stringify(cache));

const loginToAcr = () => dockerLoginUser && dockerLoginPassword && containerRegistry && spawnProcess('docker', ['login', '-u', dockerLoginUser, '-p', dockerLoginPassword, containerRegistry], { env: process.env, cwd: process.cwd() }, globalLogger)

const copyDeploymentArtifactsToOutputFolder = async (settings: BuildSettingsWithDigest[]) => {
    await createFolder(artifactfolder);
    for(const setting of settings){
        const replaceValues = {
            servicename: setting.servicename,
            image: `${containerRegistry}/${setting.servicename}@${setting.acrDigest}`,
        }
        const deploymentFileContent = await readFileasync(setting.projectPath + sep + (setting.deploymentfile || 'deployment.yaml' ))
        const replacedDeploymentContent = replaceTokens(globalLogger, deploymentFileContent, replaceValues, process.env, false)
        const serviceFolder = artifactfolder + sep + setting.servicename;
        await createFolder(serviceFolder);
        await copyFileAsync(setting.projectPath + sep + configurationFile, serviceFolder + sep + configurationFile);
        await writeFileAsync(serviceFolder + sep + artifactfile, replacedDeploymentContent);
    }
    
}

export const buildAndPush = async () => {
    await del(artifactfolder);
    const digestCache = await getDigestCache();
    await loginToAcr()
    const configurationFiles = await findConfigurationFiles(sourcesDirectory)
    const builtAndPushedImages = await buildAndPushImages(configurationFiles, digestCache);
    await copyDeploymentArtifactsToOutputFolder(builtAndPushedImages);
    await persistDigestCache(digestCache);
}
