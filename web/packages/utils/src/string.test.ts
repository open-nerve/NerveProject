// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { isCommentEmpty, isEmptyHtmlString, stripAndTruncateHTML } from "./string";

// The HTML helpers read HTML the way the browser parses it. The notification preview shows the text of a
// comment: entities come out decoded (sanitize-html, which these helpers used before, returned them escaped,
// and the preview showed "Tom &amp; Jerry").

describe("stripAndTruncateHTML", () => {
  it("returns the text, with entities decoded", () => {
    expect(stripAndTruncateHTML("<p>Tom &amp; Jerry</p>")).toBe("Tom & Jerry");
    expect(stripAndTruncateHTML("<p>a &lt; b</p><p>c&nbsp;d</p>")).toBe("a < bc d");
  });

  it("trims the text and cuts it at the given length", () => {
    expect(stripAndTruncateHTML("<p>  hello world  </p>", 5)).toBe("hello...");
    expect(stripAndTruncateHTML("<p></p>")).toBe("");
  });
});

// What the editor writes for a description that holds only an image, or only a mention. The draft modal asks
// isEmptyHtmlString whether a new work item's description is empty (an empty one closes without asking), and
// isCommentEmpty asks it for a comment: both count the same tags as content.
const IMAGE_ONLY =
  '<image-component src="asset-1" width="35%" alignment="center" status="uploaded"></image-component><p></p>';
const MENTION_ONLY =
  '<p><mention-component id="mention-1" entity_identifier="user-1" entity_name="user_mention"></mention-component></p>';

describe("isEmptyHtmlString", () => {
  it("is empty when there is no text", () => {
    expect(isEmptyHtmlString("<p></p>")).toBe(true);
    expect(isEmptyHtmlString("<p>&nbsp;</p><p><br></p>")).toBe(true);
    expect(isEmptyHtmlString("<p>x</p>")).toBe(false);
  });

  it("does not call an image-only or mention-only description empty", () => {
    expect(isEmptyHtmlString(IMAGE_ONLY)).toBe(false);
    expect(isEmptyHtmlString(MENTION_ONLY)).toBe(false);
    expect(isEmptyHtmlString('<p></p><img src="x">')).toBe(false);
  });
});

describe("isCommentEmpty", () => {
  it("counts an image or a mention as content", () => {
    expect(isCommentEmpty(undefined)).toBe(true);
    expect(isCommentEmpty("<p></p>")).toBe(true);
    expect(isCommentEmpty("  ")).toBe(true);
    expect(isCommentEmpty(IMAGE_ONLY)).toBe(false);
    expect(isCommentEmpty(MENTION_ONLY)).toBe(false);
  });
});
