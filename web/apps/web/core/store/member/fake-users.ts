/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A user as nerve gives him with a membership, of a workspace or of a project, for the tests of the members' stores and
// pages. It builds no store, so that the projects' fakes give him too without the account's store, which imports the
// tab's session.

import type { MemberUser } from "@nerve/api-client";

/** A user as nerve gives him with his membership: the name names him. */
export function userOf(name: string): MemberUser {
  return {
    id: `u-${name}`,
    display_name: name,
    first_name: name,
    last_name: "Doe",
    avatar_url: null,
    email: `${name}@example.com`,
  };
}
