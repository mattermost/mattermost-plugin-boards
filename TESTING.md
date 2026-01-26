# Testing Guide

## Running Tests

### Run all tests
```bash
cd webapp
npm test
```

### Run specific test files
```bash
cd webapp
npm test -- --testPathPattern="markdown"
npm test -- --testPathPattern="imagePaste"
npm test -- --testPathPattern="cardDialog"
```

### Update snapshots
```bash
cd webapp
npm run updatesnapshot
```

### Update snapshots for specific tests
```bash
cd webapp
npm run updatesnapshot -- --testPathPattern="markdown"
```

## New Tests Added

### 1. Markdown Block Tests
**File:** `webapp/src/components/blocksEditor/blocks/markdown/markdown.test.tsx`

Tests:
- Display snapshot
- Input snapshot
- Rendering markdown with bold text
- Rendering markdown with code blocks
- Rendering markdown with lists
- Showing placeholder when empty

**Snapshot file:** `webapp/src/components/blocksEditor/blocks/markdown/__snapshots__/markdown.test.tsx.snap`

### 2. Image Paste Tests
**File:** `webapp/src/components/cardDetail/imagePaste.test.tsx`

Tests:
- Inserting image at the end when no editing context
- Inserting image after current block when editing context provided
- Creating text block after image and calling onImageInserted callback

### 3. Card Dialog Cleanup Tests
**File:** `webapp/src/components/cardDialog.test.tsx`

New tests added:
- Cleanup empty blocks on close
- Not cleanup divider blocks even if empty

## Test Utilities Updated

### TestBlockFactory
**File:** `webapp/src/test/testBlockFactory.ts`

Added:
- `createMarkdown(card: Card): MarkdownBlock` - Factory method for creating markdown blocks in tests

## Expected Test Results

All tests should pass. If snapshots need updating, run:
```bash
cd webapp
npm run updatesnapshot
```

## Troubleshooting

### Tests timeout
If tests are taking too long, try running specific test files instead of all tests.

### Snapshot mismatches
If you see snapshot mismatches, review the changes carefully. If they are expected (due to new functionality), update the snapshots with:
```bash
npm run updatesnapshot
```

### Missing dependencies
If you see module not found errors, ensure all dependencies are installed:
```bash
cd webapp
npm install
```

