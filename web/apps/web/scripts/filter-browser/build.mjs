import { fileURLToPath } from "node:url";
import path from "node:path";
import { build } from "vite";
import tailwind from "@tailwindcss/postcss";

const root = fileURLToPath(new URL(".", import.meta.url));
const outDir = process.argv[2];
if (!outDir || !path.isAbsolute(outDir)) throw new Error("Pass the absolute temporary output directory");
await build({
  root,
  configFile: false,
  logLevel: "warn",
  resolve: { alias: { "@": path.resolve(root, "../../core") }, dedupe: ["react", "react-dom", "@headlessui/react"] },
  css: { postcss: { plugins: [tailwind()] } },
  build: { outDir, emptyOutDir: true, target: "esnext" },
});
