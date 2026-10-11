/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { DeleteApiTokenModal } from "./delete-token-modal";

// What the dialog that revokes an API token does once nerve has settled the revocation (M2 design 7.7, M3 design 7.1):
// the dialog renders on the server with a stand-in for its confirmation's dialog (fake-controls.ts), which keeps the
// props it was given; the test presses its button, as a person would, and nerve answers when the test says. Its
// session is fake-tab.ts's.

const tokens = vi.hoisted(() => ({ revokeToken: vi.fn() }));
vi.mock("@/hooks/store/user", () => ({ useApiTokens: () => tokens }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const onClose = vi.fn();

/** Renders the dialog of the token t-deploy, and presses its button; settles once the dialog has followed nerve. */
function confirmed(): Promise<unknown> {
  emptyShown();
  renderToStaticMarkup(<DeleteApiTokenModal isOpen onClose={onClose} tokenId="t-deploy" />);
  const [alert] = shown.alerts;
  if (!alert) throw new Error("no dialog");
  return Promise.resolve(alert.handleSubmit());
}

beforeEach(() => {
  signedIn();
  tokens.revokeToken.mockReset();
  tokens.revokeToken.mockResolvedValue(undefined);
  onClose.mockReset();
  toasts.length = 0;
});

describe("DeleteApiTokenModal", () => {
  it("revokes the token once its dialog is confirmed; once nerve has, says so and closes", async () => {
    await confirmed();
    expect(tokens.revokeToken.mock.calls).toEqual([["t-deploy"]]);
    expect([toasts.map((toast) => toast.type), onClose.mock.calls.length]).toEqual([["success"], 1]);
  });

  it("says nerve's reason when it refuses, and stays open", async () => {
    tokens.revokeToken.mockRejectedValueOnce(refusal(404, "identity.api_token_not_found"));
    await confirmed();
    expect([toasts, onClose.mock.calls]).toEqual([
      [
        {
          type: "error",
          title: "workspace_settings.settings.api_tokens.delete.error.title",
          message: "errors.api_token_not_found",
        },
      ],
      [],
    ]);
  });

  it.each(lateSettlings)(
    "says nothing, nor closes, when the revocation $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const revocation = heldChange<undefined>();
      tokens.revokeToken.mockReturnValueOnce(revocation.sent);
      const followed = confirmed();
      switchAccount();
      settle(revocation);
      await followed;
      expect([toasts, onClose.mock.calls]).toEqual([[], []]);
    }
  );
});
