import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.messangermax.app',
  appName: 'MessangerMax',
  webDir: 'dist',
  server: {
    url: 'http://216.162.44.70:8080',
    cleartext: true,
  },
};

export default config;
