/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { isValidNextPath, signInPath } from "./url";

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

  it("accepts a path with a query and a fragment", () => {
    expect(isValidNextPath("/settings/profile/general?tab=x#y")).toBe(true);
  });

  it("accepts a path surrounded by spaces", () => {
    expect(isValidNextPath("  /dashboard  ")).toBe(true);
  });

  // `%09` and `%0A` in the address arrive as a tab and a newline. Control characters are refused wherever
  // they are (M2 design 3.18): the browser drops tabs and newlines from an address, so "/\t/evil.example"
  // would be the protocol-relative "//evil.example".
  it.each([
    ["a tab before the path", "\t/dashboard"],
    ["a newline before the path", "\n/dashboard"],
    ["a tab between the slashes", "/\t/evil.example"],
    ["a NUL", "/dash\u0000board"],
    ["a U+001F", "/dashboard\u001f"],
    ["a DEL", "/dash\u007fboard"],
  ])("rejects a path with %s", (_, path) => {
    expect(isValidNextPath(path)).toBe(false);
  });

  it("accepts the characters around the control ranges", () => {
    expect(isValidNextPath("/a b~")).toBe(true);
    expect(isValidNextPath("/\u0080é")).toBe(true);
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

  it("rejects a slash and a backslash, which browsers read as a protocol-relative address", () => {
    expect(isValidNextPath("/\\evil.example")).toBe(false);
  });
});

describe("signInPath", () => {
  it("encodes the path, its query and its fragment as one value", () => {
    expect(signInPath("/settings/profile/general?tab=x#y")).toBe(
      "/?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y"
    );
  });

  it("gives back the path, query and fragment unchanged as next_path", () => {
    const path = "/settings/profile/general?tab=a&b=c d#y%20z";
    const next = new URL(signInPath(path), "http://nerve.test").searchParams.get("next_path");
    expect(next).toBe(path);
    expect(isValidNextPath(next ?? "")).toBe(true);
  });
});
