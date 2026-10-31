/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// jsdom does no layout: an element's getClientRects() is an empty list and its getBoundingClientRect() a
// zero rect, but a Range has neither method. ProseMirror measures a Range to scroll the selection into view
// (coordsAtPos), which TipTap's focus command asks for. A Range gets the same no-layout answers.
const unlaidOut = document.createElement("div");
// oxlint-disable-next-line no-extend-native -- declares what the test environment lacks; tests only
Range.prototype.getClientRects = () => unlaidOut.getClientRects();
// oxlint-disable-next-line no-extend-native -- declares what the test environment lacks; tests only
Range.prototype.getBoundingClientRect = () => unlaidOut.getBoundingClientRect();

// jsdom has no ClipboardEvent. ProseMirror's pasteHTML, which the editor's paste handler calls, creates one to stand
// for the paste it runs; that event carries no clipboard data.
globalThis.ClipboardEvent ??= class extends Event {
  readonly clipboardData = null;
} as unknown as typeof ClipboardEvent;
