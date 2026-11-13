// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { useForm } from "react-hook-form";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useRefusedFocus } from "@/lib/refused-focus";
import type { ModuleFormValues } from "../form";
import { ModuleStatusSelect } from "./status";

// A module's status as one of its form's fields (P6a spec 2.7, M3 closeout handoff 3): required, and while its rule
// fails it says so in a message of its own, which describes its select, refused (aria-invalid), and takes the focus as
// the form's other refused fields do (useRefusedFocus). The form here is a bare one around it, with a name field before
// it that has the focus as it is sent.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const sent = vi.fn();
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  sent.mockReset();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

/** A form of the status alone, which starts with the status given (none: no status at all). */
function Form({ status }: { status?: ModuleFormValues["status"] }) {
  const {
    control,
    handleSubmit,
    formState: { submitCount },
  } = useForm<ModuleFormValues>({ defaultValues: { status }, shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  return (
    <form ref={formRef} onSubmit={(event) => void handleSubmit(sent)(event)}>
      <input name="name" />
      <ModuleStatusSelect control={control} />
    </form>
  );
}

async function render(status?: ModuleFormValues["status"]) {
  await act(async () => root.render(<Form status={status} />));
  await act(async () => host.querySelector<HTMLInputElement>('input[name="name"]')!.focus());
}
async function submit() {
  await act(async () =>
    host.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await act(async () => vi.advanceTimersByTimeAsync(10));
}
const select = () => host.querySelector<HTMLButtonElement>("button")!;

it("says a status is required while it has none, refused, described by that and focused", async () => {
  await render();
  await submit();
  expect(sent).not.toHaveBeenCalled();
  expect(select().getAttribute("aria-invalid")).toBe("true");
  const described = select().getAttribute("aria-describedby") ?? "";
  expect(document.getElementById(described)?.textContent).toBe("common.errors.required");
  expect(document.activeElement).toBe(select());
});

it("says nothing of a status it has: not refused, described by nothing, in the natural Tab order", async () => {
  await render("planned");
  await submit();
  expect(sent).toHaveBeenCalledOnce();
  expect(select().textContent).toBe("project_modules.status.planned");
  expect([select().getAttribute("aria-invalid"), select().getAttribute("aria-describedby")]).toEqual([null, null]);
  expect(select().tabIndex).toBe(0);
  expect(host.textContent).not.toContain("common.errors.required");
});
