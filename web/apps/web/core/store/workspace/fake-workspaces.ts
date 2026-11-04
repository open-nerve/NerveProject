/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The caller's workspaces, for the tests of the stores that read them, against a fake nerve.

import type { Workspace } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { answered, json } from "@/lib/auth/fake-nerve";
import type { IWorkspaceRootStore } from "@/store/workspace";

/** A workspace of the caller's, as nerve lists it: the slug names it; a member's, unless fields say otherwise. */
export function workspaceOf(slug: string, fields: Partial<Workspace> = {}): Workspace {
  return {
    id: `id-${slug}`,
    name: slug,
    slug,
    organization_size: null,
    timezone: "UTC",
    logo_asset_id: null,
    logo_url: null,
    logo_url_expires_at: null,
    role: 15,
    total_members: 1,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** The store fetches the caller's workspaces, and nerve lists these. */
export function loadWorkspaces(nerve: FakeNerve, store: IWorkspaceRootStore, workspaces: Workspace[]) {
  return answered(
    nerve,
    () => store.fetchWorkspaces(),
    ["GET", "/api/v0/workspaces"],
    { data: workspaces },
    "the list"
  );
}

/** Mounted views may start other reads as roles change; answer only this directory and keep their responder. */
export async function refreshWorkspacesWhileMounted(
  nerve: FakeNerve,
  store: IWorkspaceRootStore,
  workspaces: Workspace[]
) {
  const previous = nerve.respond;
  nerve.respond = (call) => {
    if (call.method === "GET" && call.path === "/api/v0/workspaces") call.answer(json(200, { data: workspaces }));
    else previous?.(call);
  };
  try {
    return await store.fetchWorkspaces();
  } finally {
    nerve.respond = previous;
  }
}
