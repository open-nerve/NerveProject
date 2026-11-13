/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { refusal } from "@/lib/fake-refusal";
import { formRefusal, type FormRefusal } from "./form-refusal";

// Where a form shows nerve's refusal of a list it sent (a module's members, P6a spec 2.7): nerve names each element it
// refuses by its place in the list (member_ids[2]), and the form shows it under the list's own field. The rest of the
// decision is projectRefusal's tests' (project-refusal.test.ts).

const FIELDS = ["name", "member_ids"] as const;

describe("formRefusal, of a list", () => {
  it.each<{ when: string; fields: readonly string[]; error: unknown; shows: FormRefusal<string> }>([
    {
      when: "an element refused: under the list's field",
      fields: FIELDS,
      error: refusal(422, "validation_failed", [{ field: "member_ids[2]", code: "not_allowed" }]),
      shows: { kind: "fields", fields: { member_ids: "errors.field.not_allowed" } },
    },
    {
      when: "several elements refused, with another field: the first element's message under the list's field",
      fields: FIELDS,
      error: refusal(422, "validation_failed", [
        { field: "member_ids[1]", code: "not_allowed" },
        { field: "name", code: "too_long" },
        { field: "member_ids[3]", code: "duplicate" },
      ]),
      shows: { kind: "fields", fields: { member_ids: "errors.field.not_allowed", name: "errors.field.too_long" } },
    },
    {
      when: "the list refused as a whole: under its field",
      fields: FIELDS,
      error: refusal(422, "validation_failed", [{ field: "member_ids", code: "too_long" }]),
      shows: { kind: "fields", fields: { member_ids: "errors.field.too_long" } },
    },
    {
      when: "an element of a list the form shows nothing under: a toast",
      fields: ["name"],
      error: refusal(422, "validation_failed", [{ field: "member_ids[0]", code: "not_allowed" }]),
      shows: { kind: "toast" },
    },
  ])("shows $when", ({ fields, error, shows }) => {
    expect(formRefusal(error, fields, {})).toEqual(shows);
  });
});
