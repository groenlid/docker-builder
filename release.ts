import { findConfigurationFiles } from './build';
import { sep } from 'path';
import { replaceTokens, writeFileAsync, readFileasync } from './lib/helpers';
import { createLogger } from './lib/logger';
import { EOL } from 'os';
import { artifactfile } from './lib/constants';
import { BuildSettingsWithProjectPath } from './lib/config';

const globalLogger = createLogger("release-service")
const skipdeploymentKey = "nodeploy"

const getEnvironmentVariablesForService = (setting: BuildSettingsWithProjectPath): Record<string,string> => {
    const envVariablesForService: Record<string, string> = {};
    const servicename = setting.servicename.toLocaleUpperCase();
    const combinerTokensToReplace = ['-','_','.', ''];
    for(const [key, value] of Object.entries(process.env)) {
        if(!key.startsWith(servicename))
            continue;
        if(!value)
            continue;
        const replaced = combinerTokensToReplace.reduce((cur, next) => cur.replace(servicename + next, ''), key);
        envVariablesForService[replaced] = value;
    }

    return envVariablesForService;
}

const artifactfolder = process.env["SYSTEM_ARTIFACTSDIRECTORY"] || '.';

export const prepareForRelease = async () => {

    const configurationFiles = await findConfigurationFiles(artifactfolder);
    // Replace correct stuff based on the branch it's built from..
    let yamlcontent: Record<string, string[]> = {};

    for(const setting of configurationFiles) {
        const deploymentFile = setting.projectPath + sep + artifactfile;
        const deploymentFileContent = await readFileasync(deploymentFile);
        const deploymentFilePath = artifactfolder + sep + setting.cluster + '_' + artifactfile;

        const replaceValues = getEnvironmentVariablesForService(setting);
        if(!!replaceValues[skipdeploymentKey]) continue;

        yamlcontent[deploymentFilePath] = [...(yamlcontent[deploymentFilePath] || []), replaceTokens(globalLogger, deploymentFileContent, replaceValues, process.env, true)]
    }
    
    // Concatinate the deployment.yaml files into two yaml files separated by EOL + --- + EOL
        // One for controllers and one for serviecs....
    for(const [path, values] of Object.entries(yamlcontent)){
        globalLogger.info(`Writing to file ${process.cwd()}/${path}`);
        await writeFileAsync(path, values.join(EOL + '---' + EOL));
    }
}