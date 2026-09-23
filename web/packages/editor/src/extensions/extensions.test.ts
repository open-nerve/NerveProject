/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Extensions } from "@tiptap/core";
import { describe, expect, it } from "vitest";
// local imports
import { TOOLBAR_ITEMS } from "@/constants/common";
import type { TExtensions, TFileHandler, TMentionHandler } from "@/types";
import { CoreEditorExtensions } from "./extensions";

// What the two editor tasks of M1/P2 changed, asserted on the extension list every editor installs:
// collaboration is gone, undo and redo stay local, the nodes the kept editors need are still there,
// and the toolbar has one set of items now that the page editor is gone.

const fileHandler: TFileHandler = {
  assetsUploadStatus: {},
  cancel: () => {},
  checkIfAssetExists: async () => true,
  delete: async () => {},
  getAssetDownloadSrc: async (path) => path,
  getAssetSrc: async (path) => path,
  restore: async () => {},
  upload: async () => "asset-id",
  duplicate: async () => "asset-id",
  validation: { maxFileSize: 5 * 1024 * 1024 },
};

const mentionHandler: TMentionHandler = { renderComponent: () => null };

type TOverrides = { editable?: boolean; enableHistory?: boolean; disabledExtensions?: TExtensions[] };

const buildExtensions = (overrides: TOverrides = {}): Extensions =>
  CoreEditorExtensions({
    disabledExtensions: overrides.disabledExtensions ?? [],
    editable: overrides.editable ?? true,
    enableHistory: overrides.enableHistory ?? true,
    fileHandler,
    getEditorMetaData: () => ({ file_assets: [], user_mentions: [] }),
    mentionHandler,
  });

const names = (overrides?: TOverrides) => buildExtensions(overrides).map((extension) => extension.name);

const starterKitOf = (overrides?: TOverrides) =>
  buildExtensions(overrides).find((extension) => extension.name === "starterKit");

describe("the extensions every editor installs", () => {
  it("installs no collaboration extension", () => {
    expect(names()).not.toContain("collaboration");
    expect(names()).not.toContain("collaborationCursor");
  });

  it("keeps undo and redo local, in the history of the starter kit", () => {
    expect(starterKitOf()?.options.history).not.toBe(false);
  });

  it("switches the history off when the caller asks it to", () => {
    expect(starterKitOf({ editable: false, enableHistory: false })?.options.history).toBe(false);
  });

  it("keeps the nodes the description, the comments and the history view need", () => {
    for (const name of ["mention", "imageComponent", "table", "markdown", "uniqueID", "emoji"]) {
      expect(names()).toContain(name);
    }
  });

  it("leaves the image out when the caller disables it", () => {
    expect(names()).toContain("imageComponent");
    expect(names({ disabledExtensions: ["image"] })).not.toContain("imageComponent");
  });
});

describe("the toolbar", () => {
  it("has one set of items, since one editor type is left", () => {
    expect(Object.keys(TOOLBAR_ITEMS)).toEqual(["basic", "alignment", "list", "userAction", "complex"]);
  });

  it("offers no item that only the deleted page editor showed", () => {
    const itemKeys = Object.values(TOOLBAR_ITEMS).flatMap((items) => items.map((item) => item.itemKey));
    for (const pageOnly of ["text", "h1", "h2", "h3", "h4", "h5", "h6", "table"]) {
      expect(itemKeys).not.toContain(pageOnly);
    }
    expect(itemKeys).toContain("bold");
    expect(itemKeys).toContain("to-do-list");
    expect(itemKeys).toContain("image");
  });
});
