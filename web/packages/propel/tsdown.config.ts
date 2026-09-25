import { defineConfig } from "tsdown";

export default defineConfig({
  entry: [
    "src/button/index.ts",
    "src/calendar/index.ts",
    "src/card/index.ts",
    "src/charts/*/index.ts",
    "src/context-menu/index.ts",
    "src/empty-state/index.ts",
    "src/emoji-icon-picker/index.ts",
    "src/emoji-reaction/index.ts",
    "src/icon-button/index.ts",
    "src/icons/index.ts",
    "src/menu/index.ts",
    "src/pill/index.ts",
    "src/popover/index.ts",
    "src/scrollarea/index.ts",
    "src/tab-navigation/index.ts",
    "src/toast/index.ts",
    "src/tooltip/index.ts",
    "src/utils/index.ts",
  ],
  format: ["esm"],
  dts: true,
  copy: ["src/styles"],
  exports: {
    customExports: (exports) => ({
      ...exports,
      "./styles/react-day-picker.css": "./dist/styles/react-day-picker.css",
    }),
  },
  platform: "neutral",
});
