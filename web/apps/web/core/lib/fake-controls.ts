/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for the UI kit's controls a page renders (@nerve/ui's selects and modal, propel's input, button and
// switch), for the tests of those pages: a test file mocks each such module with this one, for instance
// vi.mock("@nerve/ui", () => import("@/lib/fake-controls")), renders the page on the server, and reads in `shown` the
// props each control was given. A control renders nothing, but a modal its children: its form is the page's.

import type { ReactNode } from "react";
import { Children, isValidElement } from "react";

/** A control that gives a value: an input gives its change event, a select its picked option's value. */
type Field = { onChange: (value: unknown) => void };
/** A button: its look, what a click does, and what it says. */
type Pressed = { variant?: string; onClick?: (event: { preventDefault: () => void }) => void; children?: ReactNode };
/** A select, and its options (CustomSelect.Option): each a value and a label. */
type Select = Field & { children?: ReactNode };
/** A search select: what it gives, how its list opens, the button it shows, and its options, each by its value. */
type SearchSelect = Field & {
  focusSearchOnOpen?: boolean;
  popperModifiers?: object[];
  customButton?: ReactNode;
  options?: { value: string }[];
};
/** A switch: what a flip does. */
type Toggle = { onCheckedChange: (checked: boolean) => unknown };

/** The props each control was given, in the order rendered; a test empties them before each case (emptyShown). */
export const shown: {
  inputs: Field[];
  buttons: Pressed[];
  selects: Select[];
  searchSelects: SearchSelect[];
  switches: Toggle[];
  modals: { children?: ReactNode }[];
} = { inputs: [], buttons: [], selects: [], searchSelects: [], switches: [], modals: [] };

/** The controls as a test starts: none rendered. */
export function emptyShown() {
  Object.assign(shown, { inputs: [], buttons: [], selects: [], searchSelects: [], switches: [], modals: [] });
}

export function Input(props: Field) {
  shown.inputs.push(props);
  return null;
}

export function InputGroup({ children }: { children?: ReactNode }) {
  return children;
}

export function Button(props: Pressed) {
  shown.buttons.push(props);
  return null;
}

/** A link styled as a button: no class. */
export function getButtonStyling() {
  return "";
}

export function CustomSelect(props: Select) {
  shown.selects.push(props);
  return null;
}
CustomSelect.Option = function Option(_props: { value: unknown; children?: ReactNode }) {
  return null;
};

export function CustomSearchSelect(props: SearchSelect) {
  shown.searchSelects.push(props);
  return null;
}

export function Switch(props: Toggle) {
  shown.switches.push(props);
  return null;
}

export function ModalCore(props: { children?: ReactNode }) {
  shown.modals.push(props);
  return props.children;
}

export const EModalPosition = { CENTER: "center" };
export const EModalWidth = { XXL: "xxl" };

/** Picks the option of a select whose label is label: the select gives the page that option's value, as a click would. */
export function pick(select: Select, label: string) {
  const options = Children.toArray(select.children).filter(isValidElement<{ value: unknown; children?: ReactNode }>);
  const option = options.find((element) => element.props.children === label);
  if (!option) throw new Error(`the select offers no ${label}`);
  select.onChange(option.props.value);
}

/** Submits the form the last modal rendered, as its submit button would; settles once the form's handler has. */
export async function submitModalForm() {
  const form = shown.modals.at(-1)?.children;
  if (!isValidElement<{ onSubmit: (event: object) => Promise<void> }>(form)) {
    throw new Error("the modal showed no form");
  }
  await form.props.onSubmit({ preventDefault: () => {}, persist: () => {} });
}
