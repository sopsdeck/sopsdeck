import { defineConfig } from 'astro/config';
import cloudflare from '@astrojs/cloudflare';

export default defineConfig({
  site: 'https://sopsdeck.com',
  output: 'server',
  adapter: cloudflare(),
  vite: {
    server: {
      fs: { allow: ['..'] },
    },
  },
});
