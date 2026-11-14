// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { expect, it } from "vitest";
import { until } from "@/lib/auth/fake-time";
import { create, mountNew, submit, useFormFixture, writes } from "./fake-form";

useFormFixture();
it("a new work item under a parent is created with that parent and the form's other fields, nothing more", async () => {
  await mountNew({ name: "Child", project_id: "A", description_html: "<p>Child</p>", parent_id: "parent" });
  await submit();
  await act(async () => until(() => writes().length === 1, "create"));
  expect(create).toHaveBeenCalledExactlyOnceWith("A", {
    name: "Child",
    description_html: "<p>Child</p>",
    priority: "none",
    assignee_ids: [],
    label_ids: [],
    parent_id: "parent",
  });
  expect(writes()[0]!.body).toMatchObject({ name: "Child", parent_id: "parent" });
});
