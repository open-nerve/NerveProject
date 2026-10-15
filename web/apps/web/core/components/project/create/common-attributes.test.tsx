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
import ProjectCommonAttributes from "./common-attributes";

// What a screen reader is told of a field of the project's creation form that shows a message (M3 closeout, C11-1),
// in a browser's document (jsdom): the field is invalid, its message is its description (propel's FieldError in its
// Field), and a refusal that names it, or a rule it fails on submit, gives it the focus (its Controller's ref).

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

/** The form's fields, in a FormProvider, as the creation modal renders them; its methods go to the test. */
function Harness({ onForm }: { onForm: (methods: UseFormReturn<ProjectCreationForm>) => void }) {
  const methods = useForm<ProjectCreationForm>({ defaultValues: getProjectFormValues() });
  useEffect(() => {
    onForm(methods);
  }, [methods, onForm]);
  return (
    <FormProvider {...methods}>
      <ProjectCommonAttributes
        setValue={methods.setValue}
        isMobile={false}
        shouldAutoSyncIdentifier
        setShouldAutoSyncIdentifier={() => undefined}
      />
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
    expect(described("identifier")).toMatchObject({ invalid: null, description: "" });
    await act(async () => {
      renderedForm().setError("identifier", { type: "server", message: "Taken" }, { shouldFocus: true });
    });
    const identifier = described("identifier");
    expect([identifier.invalid, identifier.description]).toEqual(["true", "Taken"]);
    expect(document.activeElement).toBe(identifier.control);
  });

  it("is invalid, described by its rule's message, and focused, when the form is sent without it", async () => {
    await act(async () => {
      await renderedForm().handleSubmit(() => undefined)();
    });
    const name = described("name");
    expect([name.invalid, name.description]).toEqual(["true", "name_is_required"]);
    expect(document.activeElement).toBe(name.control);
  });
});
