// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useEffect, useState, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EFileAssetType, type TNameDescriptionLoader } from "@nerve/types";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { ApiError } from "@/lib/api-error";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { DescriptionInput } from "@/components/editor/rich-text/description-input";
import { IssueTitleInput } from "./title-input";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({ getWorkspaceBySlug: () => ({ id: "id-ws" }) }),
}));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: (props: ComponentProps<typeof RichTextEditor>) => {
    const [value, setValue] = useState(props.initialValue);
    useEffect(() => setValue(props.initialValue), [props.initialValue]);
    return (
      <>
        <textarea
          value={value}
          onChange={(event) => {
            setValue(event.target.value);
            props.onChange?.({}, event.target.value);
          }}
        />
        <button
          onClick={() => {
            props.onChange?.({}, "undo");
            props.onChange?.({}, props.initialValue);
            setValue(props.initialValue);
          }}
        >
          undo then redo
        </button>
      </>
    );
  },
}));
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let requests: HeldChange<void>[];
const save = vi.fn<(value: string) => Promise<void>>();
const nextSave = vi.fn<(value: string) => Promise<void>>();
let mounted: boolean;
beforeEach(() => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  requests = [];
  save.mockReset().mockImplementation(() => {
    const held = heldChange<void>();
    requests.push(held);
    return held.sent;
  });
  nextSave.mockReset().mockResolvedValue();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  mounted = true;
});
afterEach(async () => {
  if (mounted) await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});
type Kind = "title" | "description";
function Harness({
  kind,
  confirmed = "original",
  submit = save,
}: {
  kind: Kind;
  confirmed?: string;
  submit?: typeof save;
}) {
  const [status, setStatus] = useState<TNameDescriptionLoader>("saved");
  return (
    <>
      <output>{status}</output>
      {kind === "title" ? (
        <IssueTitleInput
          workspaceSlug="ws"
          projectId="A"
          issueId="x"
          value={confirmed}
          setIsSubmitting={setStatus}
          issueOperations={{ update: (_ws, _project, _id, body) => submit(body.name) }}
        />
      ) : (
        <DescriptionInput
          key="x"
          workspaceSlug="ws"
          projectId="A"
          entityId="x"
          fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
          initialValue={confirmed}
          onSubmit={(body) => submit(body.description_html)}
          setIsSubmitting={setStatus}
        />
      )}
    </>
  );
}
const render = async (kind: Kind, confirmed = "original", submit = save) =>
  act(async () => root.render(<Harness kind={kind} confirmed={confirmed} submit={submit} />));
function input(value: string) {
  const textarea = host.querySelector("textarea")!;
  Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(textarea, value);
  textarea.dispatchEvent(new Event("input", { bubbles: true }));
}
const type = async (value: string) => act(async () => input(value));
const debounce = async () => {
  await act(async () => vi.advanceTimersByTimeAsync(1500));
  await act(async () => {});
};
const unmount = async () => {
  await act(async () => root.unmount());
  mounted = false;
};
for (const kind of ["title", "description"] as const) {
  it(`${kind} redoReplacesPendingUndoEvenAtConfirmedValue before the render catches up`, async () => {
    await render(kind, "confirmed");
    // An editor can emit both transactions in the same event turn. React has not committed between them.
    await act(async () => {
      if (kind === "description") host.querySelector<HTMLButtonElement>("button")!.click();
      else {
        input("undo");
        input("confirmed");
      }
    });
    await debounce();
    expect(save).not.toHaveBeenCalled();
    expect(host.querySelector("textarea")!.value).toBe("confirmed");
    expect(host.querySelector("output")!.textContent).toBe("saved");
    await unmount();
    expect(save).not.toHaveBeenCalled();
  });
  it(`${kind} ignores a late answer after the editor owner has been replaced`, async () => {
    await render(kind);
    await type("old owner");
    await debounce();
    await act(async () => root.render(<Harness key="another-owner" kind={kind} confirmed="new owner" />));
    await act(async () => requests[0]!.answer());
    expect(host.querySelector("textarea")!.value).toBe("new owner");
    expect(host.querySelector("output")!.textContent).toBe("saved");
    await debounce();
    expect(save.mock.calls).toEqual([["old owner"]]);
  });
  it(`${kind} preserves refused input, shows failure and retries only on request`, async () => {
    await render(kind);
    await type("keep this");
    await debounce();
    expect(save).toHaveBeenCalledExactlyOnceWith("keep this");
    await act(async () =>
      requests[0]!.refuse(new ApiError(403, { status: 403, code: "forbidden", title: "Forbidden" }))
    );
    expect(host.querySelector("output")!.textContent).toBe("failed");
    expect(host.querySelector("textarea")!.value).toBe("keep this");
    expect(host.querySelector('[role="alert"]')!.textContent).toContain("errors.forbidden");
    await act(async () => vi.advanceTimersByTimeAsync(5000));
    expect(save).toHaveBeenCalledTimes(1);
    await act(async () =>
      [...host.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!.click()
    );
    expect(save).toHaveBeenCalledTimes(2);
    await act(async () => requests[1]!.answer());
    expect(host.querySelector("output")!.textContent).toBe("submitted");
  });
  it(`${kind} does not let an older confirmation overwrite newer input or mark it saved`, async () => {
    await render(kind);
    await type("first");
    await debounce();
    await type("second");
    await act(async () => requests[0]!.answer());
    await render(kind, "first");
    expect(host.querySelector("textarea")!.value).toBe("second");
    expect(host.querySelector("output")!.textContent).toBe("submitting");
    await debounce();
    expect(save.mock.calls.map(([value]) => value)).toEqual(["first", "second"]);
    await act(async () => requests[1]!.answer());
    expect(host.querySelector("output")!.textContent).toBe("submitted");
  });
  it(`${kind} does not resend an in-flight write when its owner unmounts`, async () => {
    await render(kind);
    await type("once");
    await debounce();
    expect(save).toHaveBeenCalledTimes(1);
    await unmount();
    expect(save).toHaveBeenCalledTimes(1);
    await act(async () => requests[0]!.answer());
  });
  it(`${kind} uses the current submit callback for the next edit`, async () => {
    await render(kind);
    await render(kind, "original", nextSave);
    await type("new callback");
    await debounce();
    expect(nextSave).toHaveBeenCalledExactlyOnceWith("new callback");
    expect(save).not.toHaveBeenCalled();
  });
}

for (const kind of ["title", "description"] as const) {
  it(`${kind} saves a user reversion while an earlier value is still in flight`, async () => {
    await render(kind);
    await type("first");
    await debounce();
    await type("original");
    await act(async () => requests[0]!.answer());
    await render(kind, "first");
    expect(host.querySelector("textarea")!.value).toBe("original");
    await debounce();
    expect(save.mock.calls.map(([value]) => value)).toEqual(["first", "original"]);
    await act(async () => requests[1]!.answer());
  });
  it(`${kind} flushes a never-attempted last edit once on unmount`, async () => {
    await render(kind);
    await type("last edit");
    await unmount();
    expect(save).toHaveBeenCalledExactlyOnceWith("last edit");
    await act(async () => requests[0]!.answer());
  });
}
