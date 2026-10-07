/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The caller's workspaces, for the tests of the stores that read them, against a fake nerve.

import type { Workspace } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { json } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import type { IWorkspaceRootStore } from "@/store/workspace";

/** A workspace of the caller's, as nerve lists it: the slug names it; a member's, unless fields say otherwise. */
export function workspaceOf(slug: string, fields: Partial<Workspace> = {}): Workspace {
  return {
    id: `id-${slug}`,
    name: slug,
    slug,
    organization_size: null,
    timezone: "UTC",
    logo_url: null,
    role: 15,
    total_members: 1,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** The store fetches the caller's workspaces, and nerve lists these. */
export async function loadWorkspaces(nerve: FakeNerve, store: IWorkspaceRootStore, workspaces: Workspace[]) {
  const at = nerve.calls.length;
  const fetched = store.fetchWorkspaces();
  await until(() => nerve.calls.length === at + 1, "the list");
  nerve.calls[at]?.answer(json(200, { data: workspaces }));
  return settle(fetched, "the list");
}
