/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for the UI kit's controls a page renders (@nerve/ui's selects, modals and context menu, propel's input,
// button, switch, menu and tooltip), for the tests of those pages: a test file mocks each such module with this one,
// for instance vi.mock("@nerve/ui", () => import("@/lib/fake-controls")), renders the page on the server, and reads in
// `shown` the props each control was given. A control renders nothing, but ModalCore its children (its form is the
// page's), ModalTitle its heading, a menu its items, as it shows them once open, and a tooltip its label beside what it
// is the tooltip of.

import type { ReactNode } from "react";
import { Children, Fragment, createElement, isValidElement } from "react";

/** A control that gives a value: an input gives its change event, a select its picked option's value. */
type Field = { onChange: (value: unknown) => void };
/** A button: its look, what a click does, and what it says. */
type Pressed = { variant?: string; onClick?: (event: { preventDefault: () => void }) => void; children?: ReactNode };
/** A select, and its options (CustomSelect.Option): each a value and a label. */
type Select = Field & { children?: ReactNode };
/**
 * A search select: what it gives, how its list opens, the button it shows and its own button's place in the Tab
 * order, and its options, each by its value.
 */
type SearchSelect = Field & {
  focusSearchOnOpen?: boolean;
  popperModifiers?: object[];
  customButton?: ReactNode;
  tabIndex?: number;
  options?: { value: string }[];
};
/** A switch: what a flip does, whether it can be flipped, and its name. */
type Toggle = { onCheckedChange: (checked: boolean) => unknown; disabled?: boolean; "aria-label"?: string };
/** A confirmation's dialog (AlertModalCore): whether it shows, its button's press, whether it is busy, its words. */
type Alert = {
  isOpen: boolean;
  handleSubmit: () => unknown;
  isSubmitting: boolean;
  title: string;
  content: ReactNode;
  primaryButtonText?: { default: string; loading: string };
  secondaryButtonText?: string;
};
/** A context menu: its items, each by its key, with whether it shows. */
type ContextItems = { items: { key: string; shouldRender?: boolean }[] };
/** A button the page renders itself (<button>): what it is titled, and what a click does. */
type Clicked = {
  title?: string;
  onClick?: (event: { stopPropagation: () => void; preventDefault: () => void }) => void;
};

/** The props each control was given, in the order rendered; a test empties them before each case (emptyShown). */
export const shown: {
  inputs: Field[];
  buttons: Pressed[];
  selects: Select[];
  searchSelects: SearchSelect[];
  switches: Toggle[];
  modals: { children?: ReactNode }[];
  alerts: Alert[];
  menus: ContextItems[];
  menuItems: { children?: ReactNode }[];
} = {
  inputs: [],
  buttons: [],
  selects: [],
  searchSelects: [],
  switches: [],
  modals: [],
  alerts: [],
  menus: [],
  menuItems: [],
};

/** The controls as a test starts: none rendered. */
export function emptyShown() {
  Object.assign(shown, {
    inputs: [],
    buttons: [],
    selects: [],
    searchSelects: [],
    switches: [],
    modals: [],
    alerts: [],
    menus: [],
    menuItems: [],
  });
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

export function ContextMenu(props: ContextItems) {
  shown.menus.push(props);
  return null;
}

/** propel's menu, open: its items. */
export function Menu({ children }: { children?: ReactNode }) {
  return children;
}
Menu.MenuItem = function MenuItem(props: { children?: ReactNode }) {
  shown.menuItems.push(props);
  return props.children;
};

/** propel's tooltip: its label, in a span marked data-tooltip, then what it is the tooltip of. */
export function Tooltip({ label, children }: { label: ReactNode; children?: ReactNode }) {
  return createElement(Fragment, null, createElement("span", { "data-tooltip": "" }, label), children);
}

export function ModalCore(props: { children?: ReactNode }) {
  shown.modals.push(props);
  return props.children;
}

/** A dialog's title (ModalTitle): its heading. */
export function ModalTitle({ children }: { children?: ReactNode }) {
  return createElement("h3", null, children);
}

export function AlertModalCore(props: Alert) {
  shown.alerts.push(props);
  return null;
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

/**
 * The buttons the page renders itself (<button>) among what node holds, such as a menu item's children, in order: as
 * the page made them, inside the components that wrap them too (a tooltip). A test reads what each is titled, and
 * clicks one by its onClick.
 */
export function buttonsIn(node: ReactNode): Clicked[] {
  const buttons: Clicked[] = [];
  for (const child of Children.toArray(node)) {
    if (!isValidElement<Clicked & { children?: ReactNode }>(child)) continue;
    if (child.type === "button") buttons.push(child.props);
    buttons.push(...buttonsIn(child.props.children));
  }
  return buttons;
}

/** Submits the form the last modal rendered, as its submit button would; settles once the form's handler has. */
export async function submitModalForm() {
  const form = shown.modals.at(-1)?.children;
  if (!isValidElement<{ onSubmit: (event: object) => Promise<void> }>(form)) {
    throw new Error("the modal showed no form");
  }
  await form.props.onSubmit({ preventDefault: () => {}, persist: () => {} });
}
