import { BuildSettingsWithProjectPath, NodejsBuilder } from "../../lib/config";
import { sep, dirname } from "path";
import { readdirasync, replaceAll, writeFileAsync } from "../../lib/helpers";
import { dockerTemplate } from "./template";
import { tmpdir } from "os";
import { v4 } from "uuid";

interface IReplaceValues {
    INSTALL_AND_AUDIT: string;
    BUILD_COMMAND: string;
    RUN_COMMAND: string;
    NODE_VERSION: string;
}

const getInstallAndAuditCommand = async (settings: BuildSettingsWithProjectPath) => {
    const projectDir = await readdirasync(dirname(settings.projectPath));
    if (projectDir.some(file => file.isFile() && file.name === 'yarn.lock'))
        return `
            RUN yarn install
            RUN yarn audit
        `;

    if (projectDir.some(file => file.isFile() && file.name === 'package-lock.json'))
        return `
            RUN npm ci
            RUN npm audit
        `;
    
    return `
        RUN npm install
        RUN npm audit
    `
}

const getDockerFileContent = (replaceValues: IReplaceValues) => {
    let content = dockerTemplate;
    for(const entry of Object.entries(replaceValues)){
        content = replaceAll(content, `#{${entry[0]}}#`, entry[1]);
    }
    return content;
}

const getBuildArguments = async (settings: BuildSettingsWithProjectPath, builder: NodejsBuilder) => {
    const dir = tmpdir();
    const filename = v4();
    
    const INSTALL_AND_AUDIT = await getInstallAndAuditCommand(settings);

    const dockercontent = getDockerFileContent({ 
        INSTALL_AND_AUDIT,
        BUILD_COMMAND: builder.buildcommand ? "RUN " + builder.buildcommand : '',
        RUN_COMMAND: builder.runcommand,
        NODE_VERSION: builder.nodeversion
    });

    const dockerfilepath = dir + sep + filename;
    await writeFileAsync(dockerfilepath, dockercontent);  

    return {
        dockerbuildcontextpath: settings.projectPath,
        dockerfilepath
    }
}


export default {
    getBuildArguments
}