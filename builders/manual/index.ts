import { sep } from 'path';
import { BuildSettingsWithProjectPath, ManualBuilder } from '../../lib/config';
import { BuilderBuildArguments } from '../../build';
import { sourcesDirectory } from '../../lib/constants';

const getBuildArguments = async (settings: BuildSettingsWithProjectPath, builder: ManualBuilder | undefined): Promise<BuilderBuildArguments> => ({
    dockerfilepath: settings.projectPath + sep + ((builder && builder.dockerfile) || 'Dockerfile'),
    dockerbuildcontextpath: builder?.buildcontext && builder.buildcontext === 'projectdir' ? settings.projectPath : sourcesDirectory
});

export default {
    getBuildArguments
}