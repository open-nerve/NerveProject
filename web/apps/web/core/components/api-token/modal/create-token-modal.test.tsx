/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { isValidElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiTokenCreate, ApiTokenCreated } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { CreateApiTokenModal } from "./create-token-modal";

// What the dialog that creates an API token does once nerve has settled the creation (M2 design 7.7, M3 design 7.1):
// the dialog renders on the server with stand-ins for its modal (fake-controls.ts, which keeps what it shows) and its
// form; the test submits a token as the form would, with the form's way of saying nerve's refusal, and nerve answers
// the creation when the test says. Its session is fake-tab.ts's.

/** The form's submit, as the dialog gives it. */
type Submit = (data: ApiTokenCreate, refused: (error: unknown) => void) => Promise<void>;

const tokens = vi.hoisted(() => ({ createToken: vi.fn(), csvDownload: vi.fn() }));
vi.mock("@/hooks/store/user", () => ({ useApiTokens: () => ({ createToken: tokens.createToken }) }));
vi.mock("@nerve/utils", () => ({ csvDownload: tokens.csvDownload, renderFormattedDate: () => "Oct 18, 2026" }));
vi.mock("./form", () => ({ CreateApiTokenForm: () => null }));
vi.mock("./generated-token-details", () => ({ GeneratedTokenDetails: () => null }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const data: ApiTokenCreate = { label: "deploy", description: "ci", expired_at: "2026-10-18T00:00:00.000Z" };
const created: ApiTokenCreated = {
  id: "t-deploy",
  label: "deploy",
  description: "ci",
  expired_at: "2026-10-18T00:00:00.000Z",
  last_used: null,
  created_at: "2026-10-11T00:00:00.000Z",
  token: "nrv_pat_deploy",
};
const refused = vi.fn();

/** Renders the dialog, and submits data as its form would; settles once the dialog has followed the creation. */
function submitted(): Promise<void> {
  emptyShown();
  renderToStaticMarkup(<CreateApiTokenModal isOpen onClose={() => undefined} />);
  const form = shown.modals.at(-1)?.children;
  if (!isValidElement<{ onSubmit: Submit }>(form)) throw new Error("the dialog showed no form");
  return form.props.onSubmit(data, refused);
}

beforeEach(() => {
  signedIn();
  tokens.createToken.mockReset();
  tokens.createToken.mockResolvedValue(created);
  tokens.csvDownload.mockReset();
  refused.mockReset();
});

describe("CreateApiTokenModal", () => {
  it("creates the token the form gives; once nerve has, downloads its secret", async () => {
    await submitted();
    expect(tokens.createToken.mock.calls).toEqual([[data]]);
    expect([tokens.csvDownload.mock.calls, refused.mock.calls]).toEqual([
      [
        [
          { Title: "deploy", Description: "ci", Expiry: "Oct 18, 2026", "Secret key": "nrv_pat_deploy" },
          expect.stringMatching(/^secret-key-\d+$/),
        ],
      ],
      [],
    ]);
  });

  it("has the form say nerve's refusal, and downloads nothing", async () => {
    const error = refusal(422, "validation_failed", [{ field: "label", code: "too_long" }]);
    tokens.createToken.mockRejectedValueOnce(error);
    await submitted();
    expect([refused.mock.calls, tokens.csvDownload.mock.calls]).toEqual([[[error]], []]);
  });

  it.each(lateSettlingsAnswering(created))(
    "downloads nothing, nor has the form say anything, when the creation $settles after another tab moved this one",
    async ({ settle }) => {
      const creation = heldChange<ApiTokenCreated>();
      tokens.createToken.mockReturnValueOnce(creation.sent);
      const followed = submitted();
      switchAccount();
      settle(creation);
      await followed;
      expect([tokens.csvDownload.mock.calls, refused.mock.calls]).toEqual([[], []]);
    }
  );
});
