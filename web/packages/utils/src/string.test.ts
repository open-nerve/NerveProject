// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { copyTextToClipboard, isCommentEmpty, isEmptyHtmlString, stripAndTruncateHTML } from "./string";

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

// On an origin that is not secure (a server on plain http), navigator.clipboard is undefined and the copy is
// document.execCommand("copy") on a selected textarea. Its failure reaches the caller, as a rejected
// navigator.clipboard.writeText does, so that the page does not report a copy that did not happen.
describe("copyTextToClipboard without navigator.clipboard", () => {
  /** What the stand-in for execCommand("copy") does: returns true or false, or throws. */
  let outcome: "copies" | "fails" | "throws";
  /** The text selected when the command ran, for each run. */
  let selected: string[];

  beforeEach(() => {
    selected = [];
    // jsdom has neither; each is defined on the instance, and deleted after the test
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
    Object.defineProperty(document, "execCommand", {
      configurable: true,
      value: (command: string) => {
        const area = document.activeElement as HTMLTextAreaElement;
        selected.push(`${command}:${area.value.slice(area.selectionStart, area.selectionEnd)}`);
        if (outcome === "throws") throw new DOMException("The command is not supported.", "NotSupportedError");
        return outcome === "copies";
      },
    });
  });

  afterEach(() => {
    Reflect.deleteProperty(navigator, "clipboard");
    Reflect.deleteProperty(document, "execCommand");
  });

  it("rejects when the command reports that it did not copy, and leaves no textarea", async () => {
    outcome = "fails";
    await expect(copyTextToClipboard("nrv_pat_secret")).rejects.toThrow();
    expect(selected).toEqual(["copy:nrv_pat_secret"]);
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("rejects when the command throws, and leaves no textarea", async () => {
    outcome = "throws";
    await expect(copyTextToClipboard("nrv_pat_secret")).rejects.toThrow("The command is not supported.");
    expect(selected).toEqual(["copy:nrv_pat_secret"]);
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("resolves when the command copies the text, which it selected, and leaves no textarea", async () => {
    outcome = "copies";
    await expect(copyTextToClipboard("nrv_pat_secret")).resolves.toBeUndefined();
    expect(selected).toEqual(["copy:nrv_pat_secret"]);
    expect(document.querySelector("textarea")).toBeNull();
  });
});
