/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement, ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Label } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { toasts } from "@/lib/fake-toast";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import type { TLabelOperationsCallbacks } from "./create-update-label-inline";
import { CreateUpdateLabelInline } from "./create-update-label-inline";

// What the label settings' inline form sends to create or change a label, and shows when nerve refuses it (M3 design
// 3.16; v0 design 7.7). The form renders on the server with stand-ins for the name's input and the buttons, which keep
// the props they were given: the test types the name as the input would, and clicks the form's button.

type Input = { onChange: (value: string) => void };
type Submit = { variant: string; onClick: (event: { preventDefault: () => void }) => void };
const shown = vi.hoisted(() => {
  const inputs: Input[] = [];
  const buttons: Submit[] = [];
  return { inputs, buttons };
});
vi.mock("@makeplane/propel/components/input", () => ({
  Input: (props: Input) => {
    shown.inputs.push(props);
    return null;
  },
  InputGroup: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@nerve/propel/button", () => ({
  Button: (props: Submit) => {
    shown.buttons.push(props);
    return null;
  },
}));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "bug", 65535);

/** Renders the form, types the name, and clicks its button, which sends what the form holds. */
function submit(form: ReactElement, name: string) {
  shown.inputs.length = 0;
  shown.buttons.length = 0;
  renderToStaticMarkup(form);
  const [input] = shown.inputs;
  const button = shown.buttons.find((props) => props.variant === "primary");
  if (!input || !button) throw new Error("the form showed no name or no button");
  input.onChange(name);
  button.onClick({ preventDefault: () => {} });
}

/** The form for a change of a label, or for a creation with none. */
const formOf = (callbacks: TLabelOperationsCallbacks, label?: Label) => (
  <CreateUpdateLabelInline
    labelForm
    setLabelForm={vi.fn()}
    isUpdating={label !== undefined}
    labelOperationsCallbacks={callbacks}
    labelToUpdate={label}
  />
);

beforeEach(() => {
  toasts.length = 0;
});

describe("the label settings' inline form", () => {
  it("changes a label with the fields its form edits, the only ones nerve's LabelUpdate takes", async () => {
    const updateLabel = vi.fn<TLabelOperationsCallbacks["updateLabel"]>(async () => bug);
    submit(formOf({ createLabel: vi.fn(), updateLabel }, bug), "defect");
    await vi.waitFor(() => expect(updateLabel).toHaveBeenCalled());
    expect(updateLabel.mock.calls).toEqual([[bug.id, { name: "defect", color: "#F59E0B" }]]);
  });

  const refuse = async (): Promise<never> => {
    throw new ApiError(409, { status: 409, code: "project.label_name_taken", title: "Conflict" });
  };
  const forms: { change: string; form: ReactElement }[] = [
    { change: "creation", form: formOf({ createLabel: refuse, updateLabel: vi.fn() }) },
    { change: "change", form: formOf({ createLabel: vi.fn(), updateLabel: refuse }, bug) },
  ];
  it.each(forms)("shows nerve's reason when it refuses a $change: the name is taken", async ({ form }) => {
    submit(form, "BUG");
    await vi.waitFor(() => expect(toasts).toHaveLength(1));
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_label_name_taken" }]);
  });
});
