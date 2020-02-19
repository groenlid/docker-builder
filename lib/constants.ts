export const getArgumentValue = (arg: string): string | undefined => {
    const appendedToValue = `--${arg}=`;
    const argument = process.argv.find(a => a.includes(appendedToValue));
    if(!argument) return undefined;
    return argument.replace(argument, appendedToValue);
}

export const configurationFile = "buildsettings.json"
export const artifactfolder = process.env.artifactfolder || "./Buildscripts/dist"
export const buildId = process.env.BUILD_BUILDID || 'Not_Configured'
export const containerRegistry = getArgumentValue('dockerContainerRegistry');
export const dockerLoginUser = getArgumentValue('dockeruser');
export const dockerLoginPassword = getArgumentValue('dockerpassword');
export const excludedFolders = ['node_modules', 'build', 'bin']
export const artifactfile = "deployment.yaml"
export const sourcesDirectory = process.env["BUILD_SOURCESDIRECTORY"] || '.';
