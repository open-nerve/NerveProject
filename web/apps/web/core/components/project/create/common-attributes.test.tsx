// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { act, useEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectCreationForm } from "@/components/projects/create/use-create-project";
import { getProjectFormValues } from "@/components/projects/create/utils";
import { useRefusedFocus } from "@/lib/refused-focus";
import ProjectCommonAttributes from "./common-attributes";

// What a screen reader is told of a field of the project's creation form that shows a message (M3 closeout, C11-1),
// in a browser's document (jsdom): the field is invalid, its message is its description (propel's FieldError in its
// Field), and a refusal that names it, or a rule it fails on submit, gives it the focus once it is both
// (useRefusedFocus), which its focus event sees.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

declare global {
  // React's flag for a test that renders within act
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

/** No sizes to watch in jsdom, which has no ResizeObserver: the description's text area grows with none. */
class NoResize {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}

let root: Root;
let container: HTMLElement;
let form: UseFormReturn<ProjectCreationForm> | undefined;

/**
 * The form's fields, in a FormProvider and a form, as the creation modal renders them, its focus after an attempt to
 * send it as the modal's (react-hook-form's own off, useRefusedFocus); its methods go to the test.
 */
function Harness({ onForm }: { onForm: (methods: UseFormReturn<ProjectCreationForm>) => void }) {
  const methods = useForm<ProjectCreationForm>({ defaultValues: getProjectFormValues(), shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(methods.formState.submitCount);
  useEffect(() => {
    onForm(methods);
  }, [methods, onForm]);
  return (
    <FormProvider {...methods}>
      <form ref={formRef}>
        <ProjectCommonAttributes
          setValue={methods.setValue}
          isMobile={false}
          shouldAutoSyncIdentifier
          setShouldAutoSyncIdentifier={() => undefined}
        />
      </form>
    </FormProvider>
  );
}

/** The form's methods, once it has rendered. */
function renderedForm(): UseFormReturn<ProjectCreationForm> {
  if (!form) throw new Error("the form has not rendered");
  return form;
}

/** The control of id, and what its description says (the text of the elements its aria-describedby names). */
function described(id: string) {
  const control = container.querySelector(`#${id}`);
  const ids = control?.getAttribute("aria-describedby")?.split(" ") ?? [];
  const description = ids.map((each) => document.getElementById(each)?.textContent ?? "").join(" ");
  return { control, invalid: control?.getAttribute("aria-invalid"), description };
}

/** What the control of id said of itself each time it took the focus from now on, as it took it (its focus event). */
function focusesOf(id: string): { invalid: string | null | undefined; description: string }[] {
  const taken: { invalid: string | null | undefined; description: string }[] = [];
  container.querySelector(`#${id}`)?.addEventListener("focus", () => {
    const { invalid, description } = described(id);
    taken.push({ invalid, description });
  });
  return taken;
}

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal("ResizeObserver", NoResize);
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
  vi.unstubAllGlobals();
});

describe("the project's creation form, a field that shows a message", () => {
  it("is invalid, described by its message, and focused, once a refusal names it", async () => {
    await act(async () => {
      renderedForm().setValue("name", "Web");
      renderedForm().setValue("identifier", "WEB");
    });
    expect(described("identifier")).toMatchObject({ invalid: null, description: "" });
    const focuses = focusesOf("identifier");
    // sent, and refused once nerve answers
    await act(async () => {
      await renderedForm().handleSubmit(async () => {
        await Promise.resolve();
        renderedForm().setError("identifier", { type: "server", message: "Taken" });
      })();
    });
    const identifier = described("identifier");
    expect([identifier.invalid, identifier.description]).toEqual(["true", "Taken"]);
    expect(document.activeElement).toBe(identifier.control);
    expect(focuses).toEqual([{ invalid: "true", description: "Taken" }]);
  });

  it("is invalid, described by its rule's message, and focused, when the form is sent without it", async () => {
    const focuses = focusesOf("name");
    await act(async () => {
      await renderedForm().handleSubmit(() => undefined)();
    });
    const name = described("name");
    expect([name.invalid, name.description]).toEqual(["true", "name_is_required"]);
    expect(document.activeElement).toBe(name.control);
    expect(focuses).toEqual([{ invalid: "true", description: "name_is_required" }]);
  });
});
