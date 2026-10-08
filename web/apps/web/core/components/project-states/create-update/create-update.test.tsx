/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { ApiError } from "@/lib/api-error";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { StateCreate } from "./create";
import type { TStateFormData } from "./form";
import { StateUpdate } from "./update";

// What the state settings' creation and change send, and show when nerve refuses them (M3 design 3.17; v0 design
// 7.7). The page renders on the server with a stand-in for StateForm, which keeps the props it was given: the test
// submits the form's data as the form would, with the edits it says.

type Form = { data: TStateFormData; onSubmit: (formData: TStateFormData) => Promise<void> };
type Toast = { type: string; title: string; message: string };
const shown = vi.hoisted(() => {
  const forms: Form[] = [];
  const toasts: Toast[] = [];
  return { forms, toasts };
});
vi.mock("@/components/project-states", () => ({
  StateForm: (props: Form) => {
    shown.forms.push(props);
    return null;
  },
}));
vi.mock("@nerve/propel/toast", () => ({
  TOAST_TYPE: { SUCCESS: "success", ERROR: "error" },
  setToast: (toast: Toast) => {
    shown.toasts.push(toast);
  },
}));
vi.mock("@nerve/i18n", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));

const web = projectOf("WEB", "w-acme");
const doing = stateOf(web, "Doing", "started", 30000, { description: "Being worked on" });

/** Renders the page, and gives the form it showed. */
function formOf(page: ReactElement): Form {
  shown.forms.length = 0;
  renderToStaticMarkup(page);
  const [form] = shown.forms;
  if (!form) throw new Error("the page showed no form");
  return form;
}

beforeEach(() => {
  shown.toasts.length = 0;
});

describe("the state settings' creation and change", () => {
  it("changes a state with the fields its form edits, the only ones nerve's StateUpdate takes", async () => {
    const update = vi.fn<TStateOperationsCallbacks["updateState"]>(async () => doing);
    const form = formOf(<StateUpdate state={doing} updateStateCallback={update} handleClose={vi.fn()} />);
    expect(form.data).toEqual({ name: "Doing", color: "#60646C", description: "Being worked on" });

    await form.onSubmit({ ...form.data, name: "In progress" });
    expect(update.mock.calls).toEqual([
      [doing.id, { name: "In progress", color: "#60646C", description: "Being worked on" }],
    ]);
  });

  const refuse = async (): Promise<never> => {
    throw new ApiError(409, { status: 409, code: "project.state_name_taken", title: "Conflict" });
  };
  const pages: { change: string; page: ReactElement }[] = [
    {
      change: "creation",
      page: <StateCreate groupKey="started" createStateCallback={refuse} handleClose={vi.fn()} />,
    },
    { change: "change", page: <StateUpdate state={doing} updateStateCallback={refuse} handleClose={vi.fn()} /> },
  ];
  it.each(pages)("shows nerve's reason when it refuses a $change: the name is taken", async ({ page }) => {
    const form = formOf(page);
    await form.onSubmit({ ...form.data, name: "Todo" });
    expect(shown.toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_state_name_taken" }]);
  });
});
