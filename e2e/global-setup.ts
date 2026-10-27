import { FullConfig } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

async function globalSetup(config: FullConfig) {
  // Create directories for test artifacts
  const dirs = [
    path.join(__dirname, 'test-results'),
    path.join(__dirname, 'test-results/failures'),
    path.join(__dirname, 'test-results/visual'),
  ];

  for (const dir of dirs) {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
  }

  console.log('Global setup complete');
}

export default globalSetup;
