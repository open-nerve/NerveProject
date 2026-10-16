// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { act, useEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { useRefusedFocus } from "./refused-focus";

// The focus a form gives the field it shows refused (M3 closeout, C11-1), in a browser's document (jsdom): a form of
// two required fields, each invalid while it has an error and described by its message, as propel's Field makes them,
// with react-hook-form's own focus off, as the forms have it.

declare global {
  // React's flag for a test that renders within act
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

type Values = { first: string; second: string };

/** The form's fields, in the document's order. */
const FIELDS: readonly (keyof Values)[] = ["first", "second"];

/** What a control said of itself as it took the focus: whether it was invalid, and its description. */
type FocusTaken = { invalid: string | null; description: string };

let root: Root;
let container: HTMLElement;
let form: UseFormReturn<Values> | undefined;

function Harness({ onForm }: { onForm: (methods: UseFormReturn<Values>) => void }) {
  const methods = useForm<Values>({ defaultValues: { first: "", second: "" }, shouldFocusError: false });
  const { errors, submitCount } = methods.formState;
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  useEffect(() => {
    onForm(methods);
  }, [methods, onForm]);
  return (
    <form ref={formRef}>
      {FIELDS.map((name) => {
        const message = errors[name]?.message;
        return (
          <div key={name}>
            <input
              id={name}
              aria-invalid={message ? true : undefined}
              aria-describedby={message ? `${name}-message` : undefined}
              {...methods.register(name, { required: `${name} is required` })}
            />
            {message && <p id={`${name}-message`}>{message}</p>}
          </div>
        );
      })}
    </form>
  );
}

/** The form's methods, once it has rendered. */
function renderedForm(): UseFormReturn<Values> {
  if (!form) throw new Error("the form has not rendered");
  return form;
}

/** The control of the field name. */
function control(name: keyof Values): HTMLInputElement {
  const input = container.querySelector<HTMLInputElement>(`#${name}`);
  if (!input) throw new Error(`no control for ${name}`);
  return input;
}

/** What the control of name said of itself each time it took the focus from now on, as it took it (its focus event). */
function focusesOf(name: keyof Values): FocusTaken[] {
  const taken: FocusTaken[] = [];
  const input = control(name);
  input.addEventListener("focus", () => {
    const ids = input.getAttribute("aria-describedby")?.split(" ") ?? [];
    const description = ids.map((id) => document.getElementById(id)?.textContent ?? "").join(" ");
    taken.push({ invalid: input.getAttribute("aria-invalid"), description });
  });
  return taken;
}

/** Fills the fields, as typed. */
async function fill(values: Values): Promise<void> {
  await act(async () => {
    for (const name of FIELDS) renderedForm().setValue(name, values[name]);
  });
}

/** Sends the form, as its submit button does, onValid what it does once its rules pass (nothing, unless given). */
async function send(onValid: () => Promise<void> | void = () => undefined): Promise<void> {
  await act(async () => {
    await renderedForm().handleSubmit(onValid)();
  });
}

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
  await act(async () => {
    root.render(<Harness onForm={(each) => (form = each)} />);
  });
});

afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  form = undefined;
});

describe("useRefusedFocus, a form's focus after an attempt to send it", () => {
  it("gives a rule failing on submit's first field, in the document's order, the focus, invalid and described as it takes it", async () => {
    const [first, second] = [focusesOf("first"), focusesOf("second")];
    await send();
    expect([first, second]).toEqual([[{ invalid: "true", description: "first is required" }], []]);
    expect(document.activeElement).toBe(control("first"));
  });

  it("gives the first of the fields nerve refused, in the document's order whatever the order they were said in, the focus, invalid and described as it takes it", async () => {
    await fill({ first: "a", second: "b" });
    const [first, second] = [focusesOf("first"), focusesOf("second")];
    // nerve's answer comes after the submit: the refusal is said under the second field, then the first
    await send(async () => {
      await Promise.resolve();
      renderedForm().setError("second", { type: "server", message: "second taken" });
      renderedForm().setError("first", { type: "server", message: "first taken" });
    });
    expect([first, second]).toEqual([[{ invalid: "true", description: "first taken" }], []]);
    expect(document.activeElement).toBe(control("first"));
  });

  it("moves no focus when nothing is refused, nor as a field typed in refuses another", async () => {
    await fill({ first: "a", second: "b" });
    control("second").focus();
    await send();
    expect(document.activeElement).toBe(control("second"));
    // typing in the second gives the first a value its rule refuses (as a workspace's name gives its slug)
    const first = focusesOf("first");
    await act(async () => {
      renderedForm().setValue("first", "", { shouldValidate: true });
    });
    expect(control("first").getAttribute("aria-invalid")).toBe("true");
    expect([document.activeElement, first]).toEqual([control("second"), []]);
  });

  it("leaves the focus on the first field refused when it has it already, with no second focus event", async () => {
    control("first").focus();
    const first = focusesOf("first");
    await send();
    expect(control("first").getAttribute("aria-invalid")).toBe("true");
    expect([document.activeElement, first]).toEqual([control("first"), []]);
  });
});
