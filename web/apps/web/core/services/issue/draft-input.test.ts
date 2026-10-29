/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import type { IssueDraftPayload } from "@nerve/api-client";
import { draftInput } from "./draft-input";
const original: IssueDraftPayload = {
  name: "before",
  description_html: "<p>keep</p>",
  priority: "future-invalid-priority",
  assignee_ids: ["actor"],
  label_ids: ["label"],
  parent_id: "parent",
  start_date: "2026-10-01",
  target_date: "2026-10-30",
  state_id: "state",
};
it("a name-only edit sends the complete payload and retains every unedited legal draft field", () => {
  expect(draftInput({ name: "after" }, original)).toStrictEqual({ payload: { ...original, name: "after" } });
});
it("clearing scalar fields removes their keys, while empty arrays and strings remain explicit", () => {
  const result = draftInput(
    {
      parent_id: null,
      start_date: undefined,
      target_date: null,
      assignee_ids: [],
      label_ids: [],
      description_html: "",
      name: "",
    },
    original
  );
  expect(result).toStrictEqual({
    payload: {
      name: "",
      description_html: "",
      priority: original.priority,
      assignee_ids: [],
      label_ids: [],
      state_id: "state",
    },
  });
  expect(original.parent_id).toBe("parent");
});
it("project omission and clearing remain distinct, and native incomplete drafts are valid inputs", () => {
  expect(draftInput({})).toStrictEqual({ payload: {} });
  expect(draftInput({ project_id: null })).toStrictEqual({ project_id: null, payload: {} });
  expect(draftInput({ project_id: "A", name: "" })).toStrictEqual({ project_id: "A", payload: { name: "" } });
});
it("only generated payload keys survive, even if a caller supplies an issue or an obsolete draft field", () => {
  const input = {
    name: "kept",
    id: "draft",
    created_by: "owner",
    cycle_id: "cycle",
    module_ids: ["module"],
    sequence_id: 123,
  };
  expect(draftInput(input, input)).toStrictEqual({ payload: { name: "kept" } });
});
