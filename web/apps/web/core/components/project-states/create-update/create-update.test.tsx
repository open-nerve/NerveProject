/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { State } from "@nerve/api-client";
import { heldChange, settledYet, signedIn } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import type { StateUnderField } from "../use-state-changes";
import { StateCreate } from "./create";
import type { TStateFormData } from "./form";
import { StateUpdate } from "./update";

// What the state settings' creation and change send, and show when nerve refuses them (M3 design 3.17, 7.6; v0 design
// 7.7). The page renders on the server with a stand-in for StateForm, which keeps the props it was given: the test
// submits the form's data as the form would, with the edits it says, and the form's setError; the state store's
// changes are stand-ins nerve answers when the test says. Its session is fake-tab.ts's.

type Form = {
  data: TStateFormData;
  onSubmit: (formData: TStateFormData, underField: StateUnderField) => Promise<void>;
};
const shown = vi.hoisted((): { forms: Form[] } => ({ forms: [] }));
const store = vi.hoisted(() => ({ createState: vi.fn(), updateState: vi.fn() }));
vi.mock("@/components/project-states", () => ({
  StateForm: (props: Form) => {
    shown.forms.push(props);
    return null;
  },
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const doing = stateOf(web, "Doing", "started", 30000, { description: "Being worked on" });
const setError = vi.fn();
const underField: StateUnderField = (field, message) => setError(field, { message });
const handleClose = vi.fn();

/** Renders the page, and gives the form it showed. */
function formOf(page: ReactElement): Form {
  shown.forms.length = 0;
  renderToStaticMarkup(page);
  const [form] = shown.forms;
  if (!form) throw new Error("the page showed no form");
  return form;
}

const pages = [
  {
    change: "creation",
    page: <StateCreate projectId={web.id} groupKey="started" handleClose={handleClose} />,
    sent: () => store.createState,
    said: "project_settings.states.created",
  },
  {
    change: "change",
    page: <StateUpdate state={doing} handleClose={handleClose} />,
    sent: () => store.updateState,
    said: "project_settings.states.updated",
  },
];

beforeEach(() => {
  signedIn();
  store.createState.mockReset();
  store.createState.mockResolvedValue(doing);
  store.updateState.mockReset();
  store.updateState.mockResolvedValue(doing);
  setError.mockReset();
  handleClose.mockReset();
  toasts.length = 0;
});

describe("the state settings' creation and change", () => {
  it("creates a state in the page's project and the form's group, with what the form holds", async () => {
    const form = formOf(<StateCreate projectId={web.id} groupKey="started" handleClose={handleClose} />);
    expect(form.data).toEqual({ name: "", description: "", color: "#f59e0b" });
    await form.onSubmit({ ...form.data, name: "Review" }, underField);
    expect(store.createState.mock.calls).toEqual([
      [web.id, { name: "Review", description: "", color: "#f59e0b", group: "started" }],
    ]);
  });

  it("changes a state with the fields its form edits, the only ones nerve's StateUpdate takes", async () => {
    const form = formOf(<StateUpdate state={doing} handleClose={handleClose} />);
    expect(form.data).toEqual({ name: "Doing", color: "#60646C", description: "Being worked on" });
    await form.onSubmit({ ...form.data, name: "In progress" }, underField);
    expect(store.updateState.mock.calls).toEqual([
      [doing.id, { name: "In progress", color: "#60646C", description: "Being worked on" }],
    ]);
  });

  it.each(pages)("says the $change made and closes the form, the form busy until it has", async (row) => {
    const answer = heldChange<State>();
    row.sent().mockReturnValueOnce(answer.sent);
    const form = formOf(row.page);
    const submitted = form.onSubmit({ ...form.data, name: "Review" }, underField);
    const settled = settledYet(submitted);
    await Promise.resolve();
    expect([settled(), handleClose.mock.calls.length, toasts]).toEqual([false, 0, []]);
    answer.answer(doing);
    await submitted;
    expect([handleClose.mock.calls.length, toasts]).toEqual([
      1,
      [{ type: "success", title: "toast.success", message: row.said }],
    ]);
  });

  it.each(pages)("shows nerve's refusal of a $change under the name, and stays open", async ({ page, sent }) => {
    sent().mockRejectedValueOnce(refusal(409, "project.state_name_taken"));
    const form = formOf(page);
    await form.onSubmit({ ...form.data, name: "Todo" }, underField);
    expect([underFields(setError), toasts, handleClose.mock.calls]).toEqual([
      [["name", "errors.project_state_name_taken"]],
      [],
      [],
    ]);
  });
});
