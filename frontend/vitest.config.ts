import { defineConfig } from 'vitest/config'
import tsconfigPaths from './vite.config'
export default defineConfig({ ...(tsconfigPaths as object), test: { environment: 'jsdom', globals: true, setupFiles: [] } })
