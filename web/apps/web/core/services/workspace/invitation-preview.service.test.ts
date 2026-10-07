/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { InvitationPreview } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { previewInvitation } from "./invitation-preview.service";

// What an invitation's link shows whoever holds it (M3 design 7.4), against a fake nerve. The client is the caller's;
// that the web app asks with the public one, without a session's token, is use-invitation-preview.test.ts.

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("previewInvitation", () => {
  it("asks nerve what the link shows, and fails for a link that names none", async () => {
    const nerve = new FakeNerve();
    const preview: InvitationPreview = {
      id: "i-dan",
      role: 15,
      declined: false,
      workspace_name: "Acme",
      workspace_slug: "acme",
      workspace_logo_url: null,
    };
    const shown = track(previewInvitation(nerve.client(), "i-dan", "nrv_inv_dan"));
    await until(() => nerve.calls.length === 1, "the preview");
    expect(nerve.calls[0]).toMatchObject({
      method: "GET",
      path: "/api/v0/workspace-invitations/i-dan",
      query: { token: "nrv_inv_dan" },
    });
    nerve.calls[0]?.answer(json(200, preview));
    await until(() => shown.settled, "the preview");
    expect(shown.value).toEqual(preview);

    const unknown = track(previewInvitation(nerve.client(), "i-zed", "nrv_inv_zed"));
    await until(() => nerve.calls.length === 2, "the preview");
    nerve.calls[1]?.answer(problem(404, "workspace.invitation_not_found"));
    await until(() => unknown.settled, "the failure");
    expect(unknown.error).toBeInstanceOf(ApiError);
  });
});
