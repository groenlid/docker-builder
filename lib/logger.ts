
export enum LogLevel {
    debug,
    information
}
const log = (id: string, message: string) => {
    const now = new Date().toLocaleString();
    console.log(`[${now}][${id}] - ${message}`);
}

export interface ILogger {
    debug: (message: string) => void;
    info: (message: string) => void;
}

export const isDebugging = () => {
    const { BUILD_DEBUG } = process.env;
    const throughArgument = process.argv.some(arg => arg.toLocaleLowerCase().trim() === 'debug');
    return !!BUILD_DEBUG || throughArgument;
}

export const createLogger = (id: string): ILogger => {
    const debugging = isDebugging();
    return {
        debug: (message: string) => debugging && log(id, message),
        info: (message: string) => log(id, message)
    }
}
