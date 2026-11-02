import { reactRouter } from "@react-router/dev/vite";
import { defineConfig } from "vite";
import { emojibase } from "./emojibase";
import { localTransfer } from "./local-transfer";

export default defineConfig(() => ({
  build: {
    assetsInlineLimit: 0,
  },
  plugins: [reactRouter(), emojibase(), localTransfer("http://127.0.0.1:8080")],
  resolve: {
    tsconfigPaths: true,
    dedupe: ["react", "react-dom", "@headlessui/react"],
  },
  server: {
    host: "127.0.0.1",
    // Nerve: during development the Go server (`make run`) answers the API.
    proxy: {
      "/api": "http://127.0.0.1:8080",
    },
  },
}));
