// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { ApiError } from "@/lib/api-error";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf } from "@/store/issue/fake-issues";
import {
  close,
  confirmed,
  create,
  host,
  mountForm,
  mountNew,
  recordFocus,
  submit,
  tab,
  type,
  useFormFixture,
  writes,
} from "./fake-form";

useFormFixture();
it("an inverted date pair is described before focus and keeps the form without a write", async () => {
  await mountForm({
    projectId: "A",
    data: {
      name: "Dated",
      project_id: "A",
      description_html: "<p>Dated</p>",
      start_date: "2026-11-20",
      target_date: "2026-11-10",
    },
  });
  const focuses = recordFocus();
  await submit();
  expect(focuses.at(-1)).toEqual({ tag: "BUTTON", invalid: "true", description: "issue.calendar.due_before_start" });
  expect(host.querySelector("form")).not.toBeNull();
  expect(writes()).toEqual([]);
  expect(close).not.toHaveBeenCalled();
});
it("projectAndTitleRefusalsHaveDescriptions at the moment focus arrives", async () => {
  await mountForm();
  const focuses = recordFocus();
  await submit();
  expect(focuses.at(-1)).toEqual({ tag: "BUTTON", invalid: "true", description: "project_id_is_required" });
  expect(host.querySelector("#name")).toHaveProperty("value", "");
  expect(host.querySelector("#name")?.getAttribute("aria-invalid")).toBe("true");
  expect(host.textContent).toContain("title_is_required");
  expect(writes()).toEqual([]);
});
it("a server name refusal stays in the form and is described before focus, with no extra creation", async () => {
  await mountNew();
  const focuses = recordFocus();
  await submit();
  await act(async () => until(() => writes().length === 1, "create"));
  // An untouched state is the server's default, not an empty UUID sent to the API.
  expect(writes()[0]!.body).not.toHaveProperty("state_id");
  const refusal = new ApiError(422, {
    status: 422,
    code: "validation_failed",
    title: "Validation failed",
    errors: [{ field: "name", code: "invalid_format", message: "Invalid title" }],
  });
  await act(async () => writes()[0]!.answer(json(422, refusal.problem)));
  expect(focuses.at(-1)).toEqual({ tag: "INPUT", invalid: "true", description: "errors.field.invalid_format" });
  expect(host.querySelector<HTMLInputElement>("#name")!.value).toBe("New");
  expect(close).not.toHaveBeenCalled();
  expect(create).toHaveBeenCalledOnce();
});
it("a synchronous pending lock spans validation, confirmation and the consumer", async () => {
  await mountNew();
  await act(async () => {
    const form = host.querySelector("form")!;
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
  await act(async () => until(() => writes().length > 0, "first create"));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(create).toHaveBeenCalledOnce();
  expect(writes()).toHaveLength(1);
  await act(async () => writes()[0]!.answer(json(201, issueDetailOf("created"))));
  expect(confirmed).toHaveBeenCalledOnce();
  expect(close).toHaveBeenCalledOnce();
  expect(writes()).toHaveLength(1);
});
it("a failed consumer follows a confirmed creation once without inviting a second POST", async () => {
  confirmed.mockRejectedValue(new Error("consumer failed"));
  await mountNew();
  await type("#name", "Once");
  await submit();
  await act(async () => until(() => writes().length === 1, "creation"));
  await act(async () => writes()[0]!.answer(json(201, { ...issueDetailOf("created"), name: "Once" })));
  expect(close).toHaveBeenCalledOnce();
  expect(confirmed).toHaveBeenCalledOnce();
  expect(tab.app.issue.issues.getIssueById("created")?.name).toBe("Once");
  expect(writes()).toHaveLength(1);
});
