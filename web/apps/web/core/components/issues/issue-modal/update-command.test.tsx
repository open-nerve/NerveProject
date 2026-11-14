// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// An edit whose description only migrated sends it as no activity of its own (skip_activity); one that also changed
// the work item's modules is a real edit and sends both with their activity (M6 P7a spec 2.6, R3). The form, its
// description and the command are real; a stand-in of the modules' field changes the form's set as the field does.
import { act } from "react";
import { useFormContext } from "react-hook-form";
import { beforeEach, expect, it } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { click, formUI, mountExisting, original, submit, useFormFixture, writes } from "./fake-form";
import type { IssueFormValues } from "./types";

/** The modules' field as it changes the form's set: a pick adds the module "m-added". */
function ModulesField() {
  const { getValues, setValue } = useFormContext<IssueFormValues>();
  return (
    <button
      type="button"
      onClick={() => setValue("module_ids", [...getValues("module_ids"), "m-added"], { shouldDirty: true })}
    >
      add a module
    </button>
  );
}

useFormFixture();
beforeEach(() => {
  formUI.extra = ModulesField;
});

it("sends a module added beside a migrated description as an edit with its activity", async () => {
  await mountExisting("issue");
  await click("add a module");
  await click("migrate description");
  await submit();
  await act(async () => until(() => writes().length === 1, "the edit"));
  expect(writes()[0]!.body).toEqual({
    description_html: '<p data-id="migrated">Original</p>',
    module_ids: ["m-added"],
  });
  await act(async () =>
    writes()[0]!.answer(
      json(200, { ...original(), description_html: '<p data-id="migrated">Original</p>', module_ids: ["m-added"] })
    )
  );
});
