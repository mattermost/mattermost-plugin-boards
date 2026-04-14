// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.

import fs from 'fs';
import path from 'path';
import {fileURLToPath} from 'url';

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const e2eRoot = path.resolve(scriptDirectory, '..');
const testsRoot = path.resolve(e2eRoot, 'tests');

const groups = {
    'e2e-shard-1': [
        'tests/board-creation/basic-board.spec.ts',
        'tests/board-permissions/public-boards.spec.ts',
        'tests/board-permissions/board-member-roles.spec.ts',
        'tests/board-permissions/comment-permissions.spec.ts',
        'tests/board-permissions/sidebar-categories.spec.ts',
    ],
    'e2e-shard-2': [
        'tests/board-creation/template-selector.spec.ts',
        'tests/card-management/card-crud.spec.ts',
        'tests/board-views/view-switching.spec.ts',
        'tests/board-management/board-title.spec.ts',
    ],
    'e2e-shard-3': [
        'tests/board-features/view-features.spec.ts',
        'tests/board-features/card-properties.spec.ts',
        'tests/channel-management/channel-board-link.spec.ts',
    ],
};

function walkSpecFiles(dirPath) {
    const entries = fs.readdirSync(dirPath, {withFileTypes: true});
    return entries.flatMap((entry) => {
        const absolutePath = path.join(dirPath, entry.name);
        if (entry.isDirectory()) {
            return walkSpecFiles(absolutePath);
        }

        if (!entry.name.endsWith('.spec.ts')) {
            return [];
        }

        return [path.relative(e2eRoot, absolutePath).replaceAll(path.sep, '/')];
    });
}

function flattenGroupSelection(groupNames) {
    return groupNames.flatMap((groupName) => {
        const files = groups[groupName];
        if (!files) {
            throw new Error(`Unknown CI test group: ${groupName}`);
        }

        return files;
    });
}

function getAllSpecs() {
    return walkSpecFiles(testsRoot).sort();
}

function validateUniqueFiles(label, files) {
    const seen = new Set();
    const duplicates = new Set();

    for (const file of files) {
        if (seen.has(file)) {
            duplicates.add(file);
        }
        seen.add(file);
    }

    if (duplicates.size > 0) {
        throw new Error(`${label} contains duplicate files:\n${[...duplicates].sort().join('\n')}`);
    }
}

function validateExistingFiles(files) {
    const missingFiles = files.filter((file) => !fs.existsSync(path.resolve(e2eRoot, file)));
    if (missingFiles.length > 0) {
        throw new Error(`Missing files in CI groups:\n${missingFiles.join('\n')}`);
    }
}

function validateCoverage() {
    const shardGroupNames = Object.keys(groups).filter((groupName) => groupName.startsWith('e2e-shard-'));
    const selectedSpecs = flattenGroupSelection(shardGroupNames).sort();
    const actualSpecs = getAllSpecs();

    validateExistingFiles(selectedSpecs);
    validateUniqueFiles('e2e shards', selectedSpecs);

    const missingSpecs = actualSpecs.filter((spec) => !selectedSpecs.includes(spec));
    if (missingSpecs.length > 0) {
        throw new Error(`Shards are missing specs:\n${missingSpecs.join('\n')}`);
    }

    const extraSpecs = selectedSpecs.filter((spec) => !actualSpecs.includes(spec));
    if (extraSpecs.length > 0) {
        throw new Error(`Shards include unexpected specs:\n${extraSpecs.join('\n')}`);
    }
}

function printUsage() {
    console.error('Usage: node scripts/ci-test-groups.mjs <list|validate|groups> [group-name ...]');
}

function main() {
    const [, , command, ...rest] = process.argv;

    if (!command) {
        printUsage();
        process.exit(1);
    }

    switch (command) {
    case 'list': {
        if (rest.length === 0) {
            throw new Error('At least one group name is required for "list".');
        }

        const files = flattenGroupSelection(rest);
        validateExistingFiles(files);
        validateUniqueFiles('Selected group list', files);

        for (const file of files) {
            console.log(file);
        }
        break;
    }
    case 'groups':
        Object.keys(groups).sort().forEach((groupName) => {
            console.log(groupName);
        });
        break;
    case 'validate':
        validateCoverage();
        console.log('CI test groups are valid.');
        break;
    default:
        throw new Error(`Unknown command: ${command}`);
    }
}

main();
