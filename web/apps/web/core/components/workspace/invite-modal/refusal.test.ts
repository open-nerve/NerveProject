/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { FieldError } from "@nerve/api-client";
import { refusal } from "@/lib/fake-refusal";
import { invitationRefusal, type InvitationRefusal } from "./refusal";

// What the invitation form shows of nerve's refusal (M3 design 2, W4), for a form of two rows.

/** nerve's refusal of a batch, 422, naming errors. */
const refused = (...errors: Pick<FieldError, "field" | "code">[]) => refusal(422, "validation_failed", errors);

describe("invitationRefusal", () => {
  it.each<{ refusal: string; error: unknown; shown: InvitationRefusal }>([
    {
      refusal: "a member's address and one invited already, each under its row",
      error: refused(
        { field: "invitations[1].email", code: "duplicate" },
        { field: "invitations[0].email", code: "not_allowed" }
      ),
      shown: {
        kind: "rows",
        rows: [
          { index: 1, message: "workspace_settings.settings.members.modal.errors.already_invited" },
          { index: 0, message: "workspace_settings.settings.members.modal.errors.already_member" },
        ],
      },
    },
    {
      refusal: "another fault of an address, under its row",
      error: refused({ field: "invitations[0].email", code: "invalid_format" }),
      shown: { kind: "rows", rows: [{ index: 0, message: "errors.field.invalid_format" }] },
    },
    {
      refusal: "a fault of a row's role, which the form has no message for, in a toast",
      error: refused(
        { field: "invitations[0].email", code: "duplicate" },
        { field: "invitations[1].role", code: "invalid_format" }
      ),
      shown: { kind: "toast" },
    },
    {
      refusal: "a row the form does not have, in a toast",
      error: refused({ field: "invitations[2].email", code: "duplicate" }),
      shown: { kind: "toast" },
    },
    {
      refusal: "a refusal that names no field, in a toast",
      error: refusal(403, "forbidden"),
      shown: { kind: "toast" },
    },
    {
      refusal: "a failure without an answer, in a toast",
      error: new TypeError("offline"),
      shown: { kind: "toast" },
    },
  ])("shows $refusal", ({ error, shown }) => {
    expect(invitationRefusal(error, 2)).toEqual(shown);
  });
});
