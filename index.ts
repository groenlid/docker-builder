#!/usr/bin/env node

import { buildAndPush } from './build';
import { prepareForRelease } from './release';

if(process.argv.some(a => a.toLocaleLowerCase() === 'build'))
    buildAndPush()
        .catch(err => console.error(err));
else if(process.argv.some(a => a.toLocaleLowerCase() === 'release'))
    prepareForRelease()
        .catch(err => console.error(err));