/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Label } from "@nerve/api-client";
import { heldChange, pageSettled, signedIn } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { CreateUpdateLabelInline } from "./create-update-label-inline";

// What the label settings' inline form sends to create or change a label, and how it follows nerve's answer (M3
// design 3.16, 7.6; v0 design 7.7). The form renders on the server with stand-ins for the name's input and the buttons,
// which keep the props they were given: the test types the name as the input would, and clicks the form's button; the
// label store's changes are stand-ins nerve answers when the test says. The form is react-hook-form's own, its setError
// also kept for the test: what the form says under its fields. Its session is fake-tab.ts's.

const form = vi.hoisted(() => ({ setError: vi.fn() }));
vi.mock("react-hook-form", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-hook-form")>();
  return {
    ...actual,
    useForm: (...args: Parameters<typeof actual.useForm>) => {
      const methods = actual.useForm(...args);
      const setError: typeof methods.setError = (...error) => {
        form.setError(...error);
        methods.setError(...error);
      };
      return { ...methods, setError };
    },
  };
});
const store = vi.hoisted(() => ({ createLabel: vi.fn(), updateLabel: vi.fn() }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "bug", 65535);
const onClose = vi.fn();

/** Renders the form, types the name, and clicks its button, which sends what the form holds: what the button says. */
function submit(element: ReactElement, name: string) {
  emptyShown();
  renderToStaticMarkup(element);
  const [input] = shown.inputs;
  const button = shown.buttons.find((props) => props.variant === "primary");
  if (!input || !button?.onClick) throw new Error("the form showed no name or no button");
  input.onChange(name);
  button.onClick({ preventDefault: () => {} });
  return button.children;
}

/** The form for a change of a label, or for a creation with none. */
const formOf = (label?: Label) => (
  <CreateUpdateLabelInline projectId={web.id} labelToUpdate={label} onClose={onClose} />
);

beforeEach(() => {
  signedIn();
  store.createLabel.mockReset();
  store.createLabel.mockResolvedValue(bug);
  store.updateLabel.mockReset();
  store.updateLabel.mockResolvedValue(bug);
  onClose.mockReset();
  form.setError.mockReset();
  toasts.length = 0;
});

describe("the label settings' inline form", () => {
  it("creates a label in the page's project, at the top, with the name and colour it holds, its button Add", async () => {
    const said = submit(formOf(), "Feature");
    await vi.waitFor(() => expect(store.createLabel).toHaveBeenCalled());
    expect([said, store.createLabel.mock.calls]).toEqual([
      "add",
      [[web.id, { name: "Feature", color: expect.any(String) }]],
    ]);
  });

  it("changes a label with the fields its form edits, the only ones LabelUpdate takes, its button Update", async () => {
    const said = submit(formOf(bug), "defect");
    await vi.waitFor(() => expect(store.updateLabel).toHaveBeenCalled());
    expect([said, store.updateLabel.mock.calls]).toEqual(["update", [[bug.id, { name: "defect", color: "#F59E0B" }]]]);
  });

  it("renames a label made without a colour with none, as nerve holds it", async () => {
    const docs = labelOf(web, "docs", 75535, { color: "" });
    submit(formOf(docs), "guides");
    await vi.waitFor(() => expect(store.updateLabel).toHaveBeenCalled());
    expect(store.updateLabel.mock.calls).toEqual([[docs.id, { name: "guides", color: "" }]]);
  });

  it("closes once nerve has made the change, not before", async () => {
    const answer = heldChange<Label>();
    store.updateLabel.mockReturnValueOnce(answer.sent);
    submit(formOf(bug), "defect");
    await vi.waitFor(() => expect(store.updateLabel).toHaveBeenCalled());
    await pageSettled();
    expect(onClose).not.toHaveBeenCalled();
    answer.answer(bug);
    await vi.waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  });

  it.each([
    { change: "creation", form: () => formOf(), sent: () => store.createLabel },
    { change: "change", form: () => formOf(bug), sent: () => store.updateLabel },
  ])("stays open when nerve refuses a $change, its name taken said under the name, no toast", async (row) => {
    row.sent().mockRejectedValueOnce(refusal(409, "project.label_name_taken"));
    submit(row.form(), "BUG");
    await vi.waitFor(() => expect(row.sent()).toHaveBeenCalled());
    await pageSettled();
    expect([underFields(form.setError), onClose.mock.calls, toasts]).toEqual([
      [["name", "errors.project_label_name_taken"]],
      [],
      [],
    ]);
  });
});
