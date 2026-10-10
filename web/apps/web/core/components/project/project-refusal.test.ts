/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { refusal } from "@/lib/fake-refusal";
import { projectRefusal, type ProjectRefusal } from "./project-refusal";

// What a project's form shows of nerve's refusal (M3 design 2 P1, P3): the decision, in its order.

describe("projectRefusal", () => {
  it.each<{ when: string; error: unknown; shows: ProjectRefusal }>([
    {
      when: "a name taken: under the name, whatever fields the problem names",
      error: refusal(409, "project.name_taken", [{ field: "identifier", code: "too_long" }]),
      shows: { kind: "fields", fields: { name: "errors.project_name_taken" } },
    },
    {
      when: "an identifier taken: under the identifier",
      error: refusal(409, "project.identifier_taken"),
      shows: { kind: "fields", fields: { identifier: "errors.project_identifier_taken" } },
    },
    {
      when: "field errors of the form's fields alone: under them",
      error: refusal(422, "validation_failed", [
        { field: "name", code: "not_allowed" },
        { field: "identifier", code: "too_long" },
      ]),
      shows: { kind: "fields", fields: { name: "errors.field.not_allowed", identifier: "errors.field.too_long" } },
    },
    {
      when: "a field error of a field the form shows none under, with one it has: a toast",
      error: refusal(422, "validation_failed", [
        { field: "name", code: "not_allowed" },
        { field: "project_lead_id", code: "not_allowed" },
      ]),
      shows: { kind: "toast" },
    },
    { when: "a refusal of no field: a toast", error: refusal(409, "project.archived"), shows: { kind: "toast" } },
    { when: "nerve not reached: a toast", error: new Error("offline"), shows: { kind: "toast" } },
  ])("shows $when", ({ error, shows }) => {
    expect(projectRefusal(error)).toEqual(shows);
  });
});
