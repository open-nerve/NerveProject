/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import { isValidNextPath } from "./url";

// After signing in, the app goes to the `next_path` query parameter. Anyone can put a link with any
// `next_path` in front of a user, so only a path on this site may pass; everything else would be an
// open redirect. These are the examples from the function's own documentation.

describe("isValidNextPath", () => {
  it("accepts a path on this site", () => {
    expect(isValidNextPath("/dashboard")).toBe(true);
  });

  it("accepts a path with several segments", () => {
    expect(isValidNextPath("/workspace/123")).toBe(true);
  });

  it("accepts a path surrounded by whitespace", () => {
    expect(isValidNextPath("  /dashboard  ")).toBe(true);
  });

  // Two more, beyond the documented examples: `%09` and `%0A` in the address arrive as a tab and a
  // newline. The app navigates to the value it validated, trimmed, so the trim must remove these too.
  it("accepts a path after a tab", () => {
    expect(isValidNextPath("\t/dashboard")).toBe(true);
  });

  it("accepts a path after a newline", () => {
    expect(isValidNextPath("\n/dashboard")).toBe(true);
  });

  it("rejects an absolute address", () => {
    expect(isValidNextPath("https://malicious.com")).toBe(false);
  });

  it("rejects a protocol-relative address", () => {
    expect(isValidNextPath("//malicious.com")).toBe(false);
  });

  it("rejects a javascript: address", () => {
    expect(isValidNextPath("javascript:alert(1)")).toBe(false);
  });

  it("rejects an empty string", () => {
    expect(isValidNextPath("")).toBe(false);
  });

  it("rejects a path that does not start with a slash", () => {
    expect(isValidNextPath("dashboard")).toBe(false);
  });

  it("rejects a path that starts with a backslash", () => {
    expect(isValidNextPath("\\malicious")).toBe(false);
  });
});
