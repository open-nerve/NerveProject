import { defineConfig } from "tsdown";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  dts: true,
  copy: ["src/styles"],
  exports: {
    customExports: (exports) => ({
      ...exports,
      "./styles": "./dist/styles/index.css",
    }),
  },
  platform: "neutral",
});
