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

describe("isEmptyHtmlString", () => {
  it("is empty when there is no text", () => {
    expect(isEmptyHtmlString("<p></p>")).toBe(true);
    expect(isEmptyHtmlString("<p>&nbsp;</p><p><br></p>")).toBe(true);
    expect(isEmptyHtmlString("<p>x</p>")).toBe(false);
  });

  it("is not empty when an allowed tag is there without text", () => {
    expect(isEmptyHtmlString('<p></p><img src="x">')).toBe(true);
    expect(isEmptyHtmlString('<p></p><img src="x">', ["img"])).toBe(false);
    expect(isEmptyHtmlString('<mention-component label="Ada"></mention-component>', ["mention-component"])).toBe(false);
  });
});

describe("isCommentEmpty", () => {
  it("counts an image or a mention as content", () => {
    expect(isCommentEmpty("<p></p>")).toBe(true);
    expect(isCommentEmpty('<image-component src="a"></image-component>')).toBe(false);
    expect(isCommentEmpty('<p><mention-component label="Ada"></mention-component></p>')).toBe(false);
  });
});
