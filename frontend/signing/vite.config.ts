import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3000,
  },
  build: {
    rollupOptions: {
      output: {
        // Use fixed names for the output files
        entryFileNames: "app.js",
        chunkFileNames: "vendor.js",
        assetFileNames: (assetInfo) => {
          // Keep CSS files with a fixed name
          if (assetInfo.name?.endsWith('.css')) {
            return 'app.css';
          }
          // Keep other assets with their original names
          return '[name][extname]';
        }
      }
    }
  }
});
