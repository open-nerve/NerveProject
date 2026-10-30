// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, host, renderer, changed, choices, posts, open, enter } from "./fake-issue-properties";
import { act } from "react";
import { expect, it, vi } from "vitest";
import { labelOf } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { IssuePropertyLabels } from "./issue-layouts/properties/labels";
import { namedButton } from "./fake-issue-properties";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { problem } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import { IssueLabelSelect } from "./select/dropdown";
import { IssueLabel } from "./issue-detail/label/root";
it.each(choices)("$name places duplicate-name refusal under the name without attaching or closing", async (choice) => {
  await open(choice);
  await enter();
  expect(changed).not.toHaveBeenCalled();
  await act(async () => {
    posts()[0]!.answer(problem(409, "project.label_name_taken"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(document.querySelector<HTMLInputElement>("input")?.value).toBe("New name");
  expect(document.querySelector('input[aria-invalid="true"]')).not.toBeNull();
  expect(document.body.textContent).toContain("errors.project_label_name_taken");
  expect(toasts).toHaveLength(0);
  expect(changed).not.toHaveBeenCalled();
  expect(tab.app.label.getProjectLabels("A")).toEqual([]);
});
it.each(choices)("$name attaches only the confirmed label", async (choice) => {
  await open(choice);
  await enter();
  expect(changed).not.toHaveBeenCalled();
  const confirmed = labelOf(tab.project, "New name", 1);
  await act(async () => {
    posts()[0]!.answer(json(201, confirmed));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(changed).toHaveBeenCalledExactlyOnceWith([confirmed.id]);
  expect(posts()).toHaveLength(1);
});
it.each(choices)("$name ignores an old project's ordinary refusal after its scope changed", async (choice) => {
  await open(choice);
  await enter();
  await act(async () => renderer.render(choice.view("B")));
  await act(async () => {
    posts()[0]!.answer(problem(403, "forbidden"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(toasts).toHaveLength(0);
  expect(changed).not.toHaveBeenCalled();
  expect(document.body.textContent).not.toContain("errors.forbidden");
});
it("without a project, offers only fetched labels of still-visible projects with no reads", async () => {
  const held = labelOf(tab.project, "Held A", 1);
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [held] }));
  };
  expect((await settle(tab.app.label.fetchProjectLabels("A"), "held labels")).error).toBeUndefined();
  tab.nerve.calls.length = 0;
  await act(async () => renderer.render(<IssueLabelSelect projectId={undefined} value={[]} onChange={changed} />));
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  expect(document.body.textContent).toContain("Held A");
  expect(tab.nerve.calls).toHaveLength(0);
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  });
  expect(document.body.textContent).not.toContain("Held A");
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/labels"))).toHaveLength(0);
});

it.each(choices)("$name ignores a confirmed label from the previous project", async (choice) => {
  await open(choice);
  await enter();
  await act(async () => renderer.render(choice.view("B")));
  await act(async () => {
    posts()[0]!.answer(json(201, labelOf(tab.project, "Confirmed old", 1)));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(changed).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
it.each(choices)("$name sends one creation while repeated Enter presses are pending", async (choice) => {
  await open(choice);
  await enter();
  await act(async () => {
    document.querySelector("input")!.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
  });
  await act(async () => {
    posts()[0]!.answer(json(201, labelOf(tab.project, "Only once", 1)));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(posts()).toHaveLength(1);
  expect(changed).toHaveBeenCalledTimes(1);
});
it("an admin permission loss and recovery cannot revive a pending label refusal", async () => {
  await open(choices[0]!);
  await enter();
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    posts()[0]!.answer(problem(409, "project.label_name_taken"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(document.body.textContent).not.toContain("errors.project_label_name_taken");
  expect(toasts).toHaveLength(0);
});

it("a confirmed new label whose issue PATCH is refused remains unselected and preserves the query", async () => {
  await open({
    name: "detail-native",
    view: () => <IssueLabel workspaceSlug="ws" projectId="A" issueId="x" disabled={false} />,
  });
  await enter();
  const label = labelOf(tab.project, "New name", 1);
  await act(async () => posts()[0]!.answer(json(201, label)));
  await act(async () => until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "attach confirmed label"));
  expect(tab.app.issue.issues.getIssueById("x")?.label_ids).toEqual([]);
  await act(async () => tab.nerve.calls.find((call) => call.method === "PATCH")!.answer(problem(403, "forbidden")));
  await act(async () => vi.advanceTimersByTimeAsync(20));
  expect(document.querySelector<HTMLInputElement>("input")?.value).toBe("New name");
  expect(tab.app.issue.issues.getIssueById("x")?.label_ids).toEqual([]);
  expect(toasts.filter((toast) => toast.message === "errors.forbidden")).toHaveLength(1);
});

it("serializes the actual label collection controls while a confirmed replacement is pending", async () => {
  const labels = [labelOf(tab.project, "First", 1), labelOf(tab.project, "Second", 2)];
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: labels }));
  };
  await tab.app.label.fetchProjectLabels("A");
  tab.app.issue.issues.acceptRead(
    { ...issueDetailOf("x"), label_ids: labels.map((label) => label.id) },
    tab.app.issue.captureRead()
  );
  await act(async () => renderer.render(<IssueLabel workspaceSlug="ws" projectId="A" issueId="x" disabled={false} />));

  await act(async () => namedButton("First").click());
  await act(async () => until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "remove First"));
  expect(namedButton("Second").disabled).toBe(true);
  await act(async () => namedButton("Second").click());
  let writes = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes).toHaveLength(1);
  await act(async () =>
    writes[0]!.answer(
      json(200, { ...issueDetailOf("x"), label_ids: [labels[1]!.id], updated_at: "2026-10-30T10:00:01Z" })
    )
  );
  expect(namedButton("Second").disabled).toBe(false);
  await act(async () => namedButton("Second").click());
  writes = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes).toHaveLength(2);
  expect(writes[1]!.body).toMatchObject({ label_ids: [] });
  await act(async () =>
    writes[1]!.answer(json(200, { ...issueDetailOf("x"), label_ids: [], updated_at: "2026-10-30T10:00:02Z" }))
  );
  expect(tab.app.issue.issues.getIssueById("x")?.label_ids).toEqual([]);
});

it.each([0, 1, 3])("keeps the readonly label layout inert with %s selected labels", async (count) => {
  const labels = [1, 2, 3].map((id) => labelOf(tab.project, `Label ${id}`, id));
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: labels }));
  };
  await tab.app.label.fetchProjectLabels("A");
  tab.nerve.calls.length = 0;
  await act(async () =>
    renderer.render(
      <IssuePropertyLabels
        projectId="A"
        value={labels.slice(0, count).map((label) => label.id)}
        onChange={changed}
        disabled
      />
    )
  );
  const buttons = [...document.querySelectorAll("button")];
  expect(buttons.length).toBeGreaterThan(0);
  expect(buttons.every((button) => button.disabled)).toBe(true);
  await act(async () => buttons.forEach((button) => button.click()));
  expect(tab.nerve.calls).toHaveLength(0);
  expect(changed).not.toHaveBeenCalled();
});
