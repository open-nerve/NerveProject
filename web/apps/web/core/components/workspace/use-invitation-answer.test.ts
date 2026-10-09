/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useInvitationAnswer } from "./use-invitation-answer";

// What the invitation page sends for an answer to the invitation of its link, and what it does with nerve's word on it
// (M3 design 7.1, 7.4; W5): the hook runs as a plain function, with stand-ins for the store's answers, which nerve
// settles when the test says, and for the opening of a workspace. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({
  acceptInvitation: vi.fn(),
  declineInvitation: vi.fn(),
  openWorkspace: vi.fn(),
  reread: vi.fn(),
  mismatched: vi.fn(),
}));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({ acceptInvitation: page.acceptInvitation, declineInvitation: page.declineInvitation }),
}));
vi.mock("@/hooks/use-open-workspace", () => ({ useOpenWorkspace: () => page.openWorkspace }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const acme = workspaceOf("acme", { role: 15 });
/** The page's hook, with the page's follow-ups: reading the invitation again, and saying it is another address's. */
const answers = () => useInvitationAnswer({ reread: page.reread, mismatched: page.mismatched });

/** Each answer the page gives, by its store change. */
const given = [
  { answer: "an acceptance", store: page.acceptInvitation, make: () => answers().accept("i-ada", "nrv_inv_x") },
  { answer: "a decline", store: page.declineInvitation, make: () => answers().decline("i-ada", "nrv_inv_x") },
];

/** What the page did after nerve's word: opened, read again, said another address's, toasted. */
const followed = () => [page.openWorkspace.mock.calls, page.reread.mock.calls, page.mismatched.mock.calls, toasts];

beforeEach(() => {
  signedIn();
  page.acceptInvitation.mockReset();
  page.acceptInvitation.mockResolvedValue(acme);
  page.declineInvitation.mockReset();
  page.declineInvitation.mockResolvedValue(undefined);
  for (const followUp of [page.openWorkspace, page.reread, page.mismatched]) followUp.mockReset();
  toasts.length = 0;
});

describe("useInvitationAnswer", () => {
  it("accepts the invitation with the link's token, then opens the workspace nerve answers", async () => {
    await answers().accept("i-ada", "nrv_inv_x");
    expect(page.acceptInvitation.mock.calls).toEqual([["i-ada", "nrv_inv_x"]]);
    expect(followed()).toEqual([[[acme]], [], [], []]);
  });

  it("declines the invitation with the link's token, then reads it again", async () => {
    await answers().decline("i-ada", "nrv_inv_x");
    expect(page.declineInvitation.mock.calls).toEqual([["i-ada", "nrv_inv_x"]]);
    expect(followed()).toEqual([[], [[]], [], []]);
  });

  it.each(given)("says the invitation is another address's when nerve refuses $answer so", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(403, "workspace.invitation_email_mismatch"));
    await make();
    expect(followed()).toEqual([[], [], [[]], []]);
  });

  it.each(given)(
    "shows nerve's reason when it refuses $answer, and reads the invitation again",
    async ({ store, make }) => {
      store.mockRejectedValueOnce(refusal(404, "workspace.invitation_not_found"));
      await make();
      expect(followed()).toEqual([
        [],
        [[]],
        [],
        [{ type: "error", title: "toast.error", message: "errors.workspace_invitation_not_found" }],
      ]);
    }
  );

  describe.each(given)("$answer", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const answer = heldChange<undefined>();
        store.mockReturnValueOnce(answer.sent);
        const made = make();
        switchAccount();
        settle(answer);
        await made;
        expect(followed()).toEqual([[], [], [], []]);
      }
    );
  });
});
