import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.messangermax.app',
  appName: 'MessangerMax',
  webDir: 'dist',
  server: {
    url: 'http://37.112.108.28:8080',
    cleartext: true,
  },
};

export default config;
