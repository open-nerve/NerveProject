/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// jsdom has no canvas. Match its null capability result for the editor emoji probe.
// Node model tests do not have HTMLCanvasElement.
if (typeof HTMLCanvasElement !== "undefined") {
  // oxlint-disable-next-line no-extend-native -- declares what the test environment lacks; tests only
  HTMLCanvasElement.prototype.getContext = () => null;
}
