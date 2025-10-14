import { createClientStateManager } from '@redhat-cloud-services/ai-client-state';
// import { LightspeedClient } from '@redhat-cloud-services/lightspeed-client';
import { OLSClient } from './olsClient';

// Initialize state manager outside React scope (following Red Hat Cloud Services pattern)
const client = new OLSClient({
  baseUrl: `${window.location.origin}/api/proxy/plugin/genie-plugin/lightspeed/`, // Always use bridge proxy
  fetchFunction: (input, init) => fetch(input, init),
});

export const stateManager = createClientStateManager(client);

// Initialize immediately when module loads (no longer auto-creates conversations)
stateManager
  .init()
  .then(() => {
    console.log('[Genie] State manager initialized successfully');
  })
  .catch((error) => {
    console.error('[Genie] State manager initialization failed:', error);
  });
