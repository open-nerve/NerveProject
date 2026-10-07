/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { InvitationPreview } from "@nerve/api-client";
import { json, type FakeNerve } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";

// The fetch of what an invitation's link shows (M3 design 7.1, 7.4), the one fetch keyed by its inputs and not by a
// session (spec 3.9, ruling A6), with stand-ins for SWR and for the web app's clients: the hook runs as a plain
// function, outside React. What nerve is asked, and its refusal, is invitation-preview.service.test.ts.

type PreviewKey = [name: string, invitationId: string, token: string];
/** What the hook handed SWR: the key, the fetcher of the key, and SWR's configuration. */
type Handed = { key: PreviewKey | null; fetcher: (key: PreviewKey) => Promise<InvitationPreview>; config: unknown };

const fake = await vi.hoisted(async (): Promise<{ nerve: FakeNerve; handed: Handed[] }> => {
  const { FakeNerve } = await import("@/lib/auth/fake-nerve");
  return { nerve: new FakeNerve(), handed: [] };
});
vi.mock("swr", () => ({
  default: (key: Handed["key"], fetcher: Handed["fetcher"], config: unknown) => {
    fake.handed.push({ key, fetcher, config });
    return {};
  },
}));
// The public client sends to the fake as it is; a session's client puts the session's access token on each request.
vi.mock("@/lib/auth/api-client", () => ({
  publicClient: fake.nerve.client(),
  apiFor: () => {
    const api = fake.nerve.client();
    api.use({
      onRequest: ({ request }) => {
        request.headers.set("Authorization", "Bearer at-1");
        return request;
      },
    });
    return api;
  },
}));

const { useInvitationPreview } = await import("./use-invitation-preview");

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
  fake.handed.length = 0;
});
afterEach(() => {
  vi.useRealTimers();
});

describe("useInvitationPreview", () => {
  it("keys the preview by the link alone, its invitation and its token, and fetches nothing for a link without both", () => {
    useInvitationPreview("i-dan", "nrv_inv_one");
    useInvitationPreview("i-dan", "nrv_inv_two");
    useInvitationPreview("i-dan", null);
    useInvitationPreview(null, "nrv_inv_one");
    expect(fake.handed.map(({ key }) => key)).toEqual([
      ["INVITATION_PREVIEW", "i-dan", "nrv_inv_one"],
      ["INVITATION_PREVIEW", "i-dan", "nrv_inv_two"],
      null,
      null,
    ]);
    // a link that names no invitation is answered once: no retry, nor a new fetch when the tab regains focus
    expect(fake.handed.map(({ config }) => config)).toEqual(
      Array.from({ length: 4 }, () => ({ revalidateOnFocus: false, shouldRetryOnError: false }))
    );
  });

  it("asks nerve with the public client, the link's invitation and token and no session's token", async () => {
    const preview: InvitationPreview = {
      id: "i-dan",
      role: 15,
      declined: false,
      workspace_name: "Acme",
      workspace_slug: "acme",
      workspace_logo_url: null,
    };
    useInvitationPreview("i-dan", "nrv_inv_dan");
    const shown = track(Promise.all(fake.handed.map(({ key, fetcher }) => key && fetcher(key))));
    await until(() => fake.nerve.calls.length === 1, "the preview");
    expect(fake.nerve.calls[0]).toMatchObject({
      method: "GET",
      path: "/api/v0/workspace-invitations/i-dan",
      query: { token: "nrv_inv_dan" },
      authorization: null,
    });
    fake.nerve.calls[0]?.answer(json(200, preview));
    await until(() => shown.settled, "the preview");
    expect(shown.value).toEqual([preview]);
  });
});
