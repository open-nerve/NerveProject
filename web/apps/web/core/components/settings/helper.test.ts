/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { pathnameToAccessKey } from "./helper";

it.each([
  ["/ws/settings/members", { workspaceSlug: "ws", accessKey: "/settings/members" }],
  ["/ws/settings/members/", { workspaceSlug: "ws", accessKey: "/settings/members" }],
  ["/ws/settings", { workspaceSlug: "ws", accessKey: "/settings" }],
  ["/ws", { workspaceSlug: "ws", accessKey: "/" }],
])("%s gives its workspace and the settings key", (pathname, expected) => {
  expect(pathnameToAccessKey(pathname)).toEqual(expected);
});
