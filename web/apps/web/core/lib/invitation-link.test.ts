/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { invitationAuthPath, invitationLink } from "./invitation-link";

// The link an admin copies for an invitation (M3 design 7.4, 9.5): the invitation page reads its id and its token
// back from the query as they were.

describe("invitationLink", () => {
  it("opens the invitation page at the origin, with the invitation's id and token", () => {
    expect(invitationLink("https://nerve.example", { id: "i-1", token: "nrv_inv_abc-_Z9" })).toBe(
      "https://nerve.example/workspace-invitations?invitation_id=i-1&token=nrv_inv_abc-_Z9"
    );
  });

  it.each(["a+b", "a&token=b", "a b/c=?#d", "雪"])("gives back the token %s as it was", (token) => {
    const link = new URL(invitationLink("https://nerve.example", { id: "i-1", token }));
    expect([link.pathname, link.searchParams.get("invitation_id"), link.searchParams.get("token")]).toEqual([
      "/workspace-invitations",
      "i-1",
      token,
    ]);
  });
});

describe("invitationAuthPath", () => {
  it.each(["/", "/sign-up"] as const)("opens %s for the link's invitation, and comes back to the link", (path) => {
    const page = new URL(invitationAuthPath(path, { id: "i-1", token: "a&b" }), "https://nerve.example");
    expect([
      page.pathname,
      page.searchParams.get("invitation_id"),
      page.searchParams.get("token"),
      page.searchParams.get("next_path"),
    ]).toEqual([path, "i-1", "a&b", "/workspace-invitations?invitation_id=i-1&token=a%26b"]);
  });
});
