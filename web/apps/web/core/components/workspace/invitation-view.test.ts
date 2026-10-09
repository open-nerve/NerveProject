/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { InvitationPreview } from "@nerve/api-client";
import { refusal } from "@/lib/fake-refusal";
import { invitationView, type InvitationView } from "./invitation-view";

// What the invitation page shows (M3 design 7.4, W5), each case one step further down the decision than the one
// before: a page that showed a later view on an earlier case's inputs fails it. nerve's error comes with the invitation
// as SWR still holds it from an earlier read, as after a refused answer's reread: the error decides.

const acme: InvitationPreview = {
  id: "i-1",
  role: 15,
  declined: false,
  workspace_name: "Acme",
  workspace_slug: "acme",
  workspace_logo_url: null,
};
const link = { invitationId: "i-1", token: "nrv_inv_x" };
const shown = { data: acme, error: undefined };
const page = { link, preview: shown, signedIn: true, mismatched: false };

describe("invitationView", () => {
  it.each<{ when: string; inputs: Parameters<typeof invitationView>[0]; view: InvitationView }>([
    {
      when: "the link has no token",
      inputs: { ...page, link: { invitationId: "i-1", token: null } },
      view: { kind: "invalid" },
    },
    {
      when: "the link has no invitation's id",
      inputs: { ...page, link: { invitationId: null, token: "nrv_inv_x" } },
      view: { kind: "invalid" },
    },
    {
      when: "nerve knows no invitation of the link",
      inputs: { ...page, preview: { data: acme, error: refusal(404, "workspace.invitation_not_found") } },
      view: { kind: "invalid" },
    },
    {
      when: "the link's id is none",
      inputs: { ...page, preview: { data: acme, error: refusal(400, "bad_request") } },
      view: { kind: "invalid" },
    },
    {
      when: "nerve could not be asked",
      inputs: { ...page, preview: { data: acme, error: new TypeError("offline") } },
      view: { kind: "unavailable" },
    },
    {
      when: "the link's invitation is on its way",
      inputs: { ...page, preview: { data: undefined, error: undefined } },
      view: { kind: "loading" },
    },
    {
      when: "the invitation was declined",
      inputs: { ...page, preview: { data: { ...acme, declined: true }, error: undefined }, signedIn: false },
      view: { kind: "declined", invitation: { ...acme, declined: true }, token: "nrv_inv_x" },
    },
    {
      when: "no one is signed in",
      inputs: { ...page, signedIn: false, mismatched: true },
      view: { kind: "sign-in", invitation: acme, token: "nrv_inv_x" },
    },
    {
      when: "nerve found the caller's address not the invitation's",
      inputs: { ...page, mismatched: true },
      view: { kind: "mismatch", invitation: acme, token: "nrv_inv_x" },
    },
    { when: "the caller may answer it", inputs: page, view: { kind: "answer", invitation: acme, token: "nrv_inv_x" } },
  ])("shows $view.kind when $when", ({ inputs, view }) => {
    expect(invitationView(inputs)).toEqual(view);
  });
});
