/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { Editor } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { StarterKit } from "@tiptap/starter-kit";
import { afterEach, describe, expect, it } from "vitest";
// local imports
import { CustomCalloutExtensionConfig } from "./extension-config";
import { ECalloutAttributeNames } from "./types";

// Every callout attribute is typed as a string (types.ts). TipTap's default parsing of an attribute turns a
// numeric string into a number, so an emoji's code point read back from HTML used to arrive as 128161, not
// "128161"; the attributes now keep what the HTML says.

const openEditors: Editor[] = [];

afterEach(() => {
  for (const editor of openEditors.splice(0)) editor.destroy();
});

const calloutOf = (content: string) => {
  const editor = new Editor({ extensions: [StarterKit, CustomCalloutExtensionConfig], content });
  openEditors.push(editor);
  let callout: ProseMirrorNode | undefined;
  editor.state.doc.descendants((node) => {
    if (node.type.name === CustomCalloutExtensionConfig.name) callout = node;
  });
  return { editor, callout };
};

const HTML =
  '<div data-block-type="callout-component" data-logo-in-use="emoji" data-emoji-unicode="128161" ' +
  'data-emoji-url="https://example.com/1f4a1.png" data-background="light-gray"><p>Note</p></div>';

describe("the callout's attributes read from HTML", () => {
  it("keeps a numeric emoji code point as the string the HTML holds", () => {
    const { callout } = calloutOf(HTML);
    expect(callout?.attrs[ECalloutAttributeNames.EMOJI_UNICODE]).toBe("128161");
  });

  it("keeps the other attributes as the HTML has them", () => {
    const { callout } = calloutOf(HTML);
    expect(callout?.attrs[ECalloutAttributeNames.LOGO_IN_USE]).toBe("emoji");
    expect(callout?.attrs[ECalloutAttributeNames.EMOJI_URL]).toBe("https://example.com/1f4a1.png");
    expect(callout?.attrs[ECalloutAttributeNames.BACKGROUND]).toBe("light-gray");
  });

  it("falls back to the defaults for the attributes the HTML leaves out", () => {
    const { callout } = calloutOf('<div data-block-type="callout-component"><p>Note</p></div>');
    expect(callout?.attrs[ECalloutAttributeNames.EMOJI_UNICODE]).toBe("128161");
    expect(callout?.attrs[ECalloutAttributeNames.ICON_NAME]).toBeUndefined();
  });

  it("writes the code point back unchanged", () => {
    const { editor } = calloutOf(HTML);
    expect(editor.getHTML()).toContain('data-emoji-unicode="128161"');
  });
});
