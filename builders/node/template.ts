export const dockerTemplate = `
FROM node:#{NODE_VERSION}#-alpine

WORKDIR /usr/src/app
COPY package.json yarn.lock package-lock.json ./

#{INSTALL_AND_AUDIT}#

COPY / ./

#{BUILD_COMMAND}#

CMD #{RUN_COMMAND}#

`