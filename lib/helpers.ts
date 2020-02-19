import { Dirent, readFile, readdir, mkdir, writeFile, copyFile } from "fs"
import { SpawnOptionsWithoutStdio } from "child_process"
import spawn from "cross-spawn"
import { ILogger, isDebugging } from "./logger"

export const readdirasync = (path: string) => new Promise<Dirent[]>((resolve, reject) => {
    readdir(path, { withFileTypes: true }, (err, files) => {
        if(err)
            reject(err)
        else
            resolve(files)
    })
})

const fileCache = new Map<string, Promise<string>>();
export const readFileasync = (path: string) => {
    const fromCache = fileCache.get(path);
    if(fromCache) return fromCache;
    const impl = new Promise<string>((resolve, reject) => {
        readFile(path, { encoding: 'utf-8' }, (err, content) => {
            if(err)
                reject(err)
            else
                resolve(content)
        })
    })
    fileCache.set(path, impl);
    return impl;
}

export const createFolder = (path: string) => new Promise((resolve, reject) => {
    mkdir(path, (err) => {
        if(err && err.code !== 'EEXIST')
            reject(err);
        else
            resolve();
    })
})

export const containsTokens = (text: string) => {
    return !!(text.match(/\#\{(.*)\}\#/g) || []).length;
}

const upperCaseObject = <T>(obj: Record<string, T>) => {
    const envVariables: Record<string, T> = {};
    for(const [key, value] of Object.entries(obj))
        envVariables[key.toLocaleUpperCase()] = value;
    return envVariables;
}

export const replaceTokens = (logger: ILogger, text: string, obj: Record<string, string | number | undefined>, env: Record<string, string | undefined>, throwOnMissing: boolean) => {
    const replaceables = (text.match(/\#\{(.*)\}\#/g) || []);
    let content = text;
    let upperCased = upperCaseObject(obj);
    for(const toReplace of replaceables) {
        const key = toReplace.substring(2, toReplace.length - 2).toLocaleUpperCase();
        const replaceValue = upperCased[key] || upperCased[replaceAll(key, ".", "_")] || env[key] || env[replaceAll(key, ".", "_")]; 
        
        if(replaceValue !== undefined && replaceValue !== null) 
            content = replaceAll(content, toReplace, replaceValue.toString());
        else if(throwOnMissing) {
            logger.debug(`Specified variables ${JSON.stringify(upperCased)}, processenv: ${JSON.stringify(env)}`)
            throw new Error(`Could not find replacevalue for key ${key}`);
        }
    }
    return content;
}

export const replaceAll = (text: string, search: string, replacement: string) => text.split(search).join(replacement)

export const writeFileAsync = (path: string, content: string) => new Promise((resolve, reject) => {
    writeFile(path, content, (err) => {
        if(err)
            reject(err)
        else
            resolve()
    })
})

export const copyFileAsync = (fromPath: string, toPath: string) => new Promise((resolve, reject) => {
    copyFile(fromPath, toPath, (err) => {
        if(err)
            reject(err)
        else
            resolve()
    })
});

export const spawnProcess = (command: string, args: string[], options: SpawnOptionsWithoutStdio, logger: ILogger): Promise<string> => {
    const proc = spawn(command, args, options)
    return new Promise<string>((resolve, reject) => {
        let datastring = "";
        let errorstring = "";
        proc.on('close', (code) => {
            if(code === 0)
                resolve(datastring);
            else
                reject(errorstring)
        })
        proc.once('error', (err) => {
            if(!isDebugging())
                logger.info(datastring);
            reject(err);
        })
        if(proc.stdout !== null)
            proc.stdout.on('data', (d) => {
                logger.debug(d.toString())
                datastring += d.toString()
            });
        if(proc.stderr !== null)
            proc.stderr.on('data', (d) => {
                logger.info(d.toString())
                errorstring += d.toString()
            });
    })
}