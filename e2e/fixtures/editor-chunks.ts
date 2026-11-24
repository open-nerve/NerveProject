/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import { expect, type Page } from "@playwright/test";

// The work-item editor's chunks in the web app's build, and a hold on its implementation (M4 C10; M7 P5b spec 2.8, R8):
// moved from stories/issue/editor-loading.spec.ts, whose stories and H8 (page)'s close story use them.

// Identify the actual build by production code in it, not an assumed bundler filename.
// These are the editor's own suggestion plugin and ProseMirror's state implementation.
export async function editorChunks() {
  const assets = new URL("../../web/apps/web/build/client/assets/", import.meta.url);
  const files = (await readdir(assets)).filter((file) => file.endsWith(".js"));
  const chunks = await Promise.all(
    files.map(async (file) => {
      const code = await readFile(new URL(file, assets), "utf8");
      const markers = ["emojiSuggestion", "Adding different instances of a keyed plugin"].filter((marker) =>
        code.includes(marker)
      );
      return { file, markers, sha256: createHash("sha256").update(code).digest("hex") };
    })
  );
  const editor = chunks.filter(({ markers }) => markers.length > 0);
  expect(editor.flatMap(({ markers }) => markers)).toEqual(
    expect.arrayContaining(["emojiSuggestion", "Adding different instances of a keyed plugin"])
  );
  return editor;
}

export async function holdEditorImplementation(page: Page) {
  const chunks = (await editorChunks()).filter(({ markers }) =>
    markers.includes("Adding different instances of a keyed plugin")
  );
  let release!: () => void;
  const loaded = new Promise<void>((resolve) => {
    release = resolve;
  });
  const held: string[] = [];
  await page.route("**/assets/*.js", async (route) => {
    const file = new URL(route.request().url()).pathname.split("/").at(-1);
    if (chunks.some((chunk) => chunk.file === file)) {
      held.push(file!);
      await loaded;
    }
    await route.continue();
  });
  return { held, release };
}
