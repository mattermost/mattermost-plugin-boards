import { FullConfig } from '@playwright/test';

async function globalTeardown(config: FullConfig) {
  // Clean up resources after all tests
  console.log('Global teardown complete');
}

export default globalTeardown;
