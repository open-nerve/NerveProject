/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * Whether a keydown is an input method's (Chinese, say), not the page's: one while the method composes (isComposing),
 * which Chromium and Firefox give, or the one that ends the composition, which Safari gives after the composition's
 * end, isComposing false and keyCode 229, the code every browser gives a key the method takes. Takes the DOM's
 * KeyboardEvent, a React keyboard event's nativeEvent.
 */
export function isComposing(event: Pick<KeyboardEvent, "isComposing" | "keyCode">): boolean {
  return event.isComposing || event.keyCode === 229;
}
