/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The work item's dialog in jsdom, for the tests of the pages that open it (P5b, P6b): jsdom has no dialog transitions
// nor the editor, so these stand in for @nerve/ui's dialog parts and the description's editor; the form's own fields
// stay real. Each test file mocks the modules with them (vi.mock is per file). Below them, what its tests read of
// nerve's refusal under a field, and such a refusal (P7a).
import { forwardRef, useImperativeHandle, type ReactNode } from "react";
import { json } from "@/lib/auth/fake-nerve";

/** @nerve/ui's dialog parts: the dialog, its children while it is open; its title, a heading. */
export const modal = {
  ModalCore: (props: { isOpen: boolean; children: ReactNode }) =>
    props.isOpen ? <div role="dialog">{props.children}</div> : null,
  ModalTitle: (props: { children: ReactNode }) => <h3>{props.children}</h3>,
};

/** The description's editor (@/components/editor/rich-text): a plain field, always ready to be discarded. */
export const editor = {
  RichTextEditor: forwardRef(function Editor(_props, ref) {
    useImperativeHandle(ref, () => ({ isEditorReadyToDiscard: () => true, clearEditor: () => {} }));
    return <textarea aria-label="description" />;
  }),
};

/** What a form's field's button says of nerve's refusal: whether it is refused, and the message that describes it. */
export const refusedAs = (field: HTMLButtonElement | undefined) => ({
  invalid: field?.getAttribute("aria-invalid") ?? null,
  message: (field?.getAttribute("aria-describedby")?.split(/\s+/) ?? [])
    .map((id) => document.getElementById(id)?.textContent ?? "")
    .join(" "),
});

/** nerve's refusal of what a form sent, naming the field at path. */
export const fieldRefused = (path: string) =>
  json(422, {
    status: 422,
    code: "validation_failed",
    title: "",
    errors: [{ field: path, code: "not_allowed", message: "" }],
  });
