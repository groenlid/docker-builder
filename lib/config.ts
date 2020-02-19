import { Decoder, object, optional, string, union, constant } from '@mojotech/json-type-validation'

export interface ManualBuilder {
    type: 'manual',
    buildcontext?: 'root' | 'projectdir';
    dockerfile?: string;
}

export interface DotnetBuilder {
    type: 'dotnet'
    dotnetruntime?: 'runtime' | 'aspnet'
}

export interface NodejsBuilder {
    type: 'nodejs';
    nodeversion: string;
    buildcommand?: string;
    runcommand: string;
}

export interface BuildSettings {
    servicename: string;
    cluster: 'controller' | 'services';
    builder?: ManualBuilder | DotnetBuilder | NodejsBuilder;
    deploymentfile?: string;
}

const DotnetBuilderDecoder: Decoder<DotnetBuilder> = object<DotnetBuilder>({
    type: constant('dotnet'),
    dotnetruntime: optional(union(constant('runtime'), constant('aspnet')))
});

const NodejsBuilderDecoder: Decoder<NodejsBuilder> = object<NodejsBuilder>({
    type: constant('nodejs'),
    buildcommand: optional(string()),
    nodeversion: string(),
    runcommand: string()
})

const ManualBuilderDecoder: Decoder<ManualBuilder> = object<ManualBuilder>({
    type: constant('manual'),
    buildcontext: optional(union(constant('root'), constant('projectdir'))),
    dockerfile: optional(string()),
});

export const BuildDecoder: Decoder<BuildSettings> = object<BuildSettings>({
    deploymentfile: optional(string()),
    servicename: string(),
    cluster: union(constant('controller'), constant('services')),
    builder: optional(union(DotnetBuilderDecoder, ManualBuilderDecoder, NodejsBuilderDecoder))
})

export interface BuildSettingsWithProjectPath extends BuildSettings {
    projectPath: string;
}

export interface BuildSettingsWithDockerBuildCacheId extends BuildSettingsWithProjectPath {
    dockerCacheId: string;
}

export interface BuildSettingsWithDigest extends BuildSettingsWithDockerBuildCacheId {
    acrDigest: string;
}