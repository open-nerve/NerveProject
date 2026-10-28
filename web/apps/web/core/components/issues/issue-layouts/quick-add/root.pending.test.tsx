// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import type { FieldValues, UseFormProps } from "react-hook-form";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TIssue } from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
import { heldChange } from "@/lib/auth/fake-tab";
import { projectOf } from "@/store/project/fake-projects";
import { issueOf } from "@/store/issue/fake-issues";
import { QuickAddIssueRoot } from "./root";

const observed = vi.hoisted(() => ({
  route: { workspaceSlug: "ws", projectId: "A" },
  submissions: [] as Promise<unknown>[],
  toast: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => {
  const { useSyncExternalStore } = await import("react");
  return {
    ...(await original<typeof import("react-router")>()),
    useParams: () =>
      useSyncExternalStore(
        (notify) => {
          window.addEventListener("routechange", notify);
          return () => window.removeEventListener("routechange", notify);
        },
        () => observed.route
      ),
  };
});
vi.mock("react-hook-form", async (original) => {
  const real = await original<typeof import("react-hook-form")>();
  return {
    ...real,
    useForm: <T extends FieldValues>(options: UseFormProps<T>) => {
      const form = real.useForm<T>(options);
      return {
        ...form,
        handleSubmit: (...args: Parameters<typeof form.handleSubmit>) => {
          const handler = form.handleSubmit(...args);
          return (...event: Parameters<typeof handler>) => {
            const pending = handler(...event);
            observed.submissions.push(pending);
            void pending.catch(() => {});
            return pending;
          };
        },
      };
    },
  };
});
vi.mock("@nerve/propel/toast", () => ({ setPromiseToast: observed.toast }));
vi.mock("../../create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ getProjectById: (id: string) => projectOf(id, "workspace") }),
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(() => {
  observed.route = { workspaceSlug: "ws", projectId: "A" };
  observed.submissions.length = 0;
  observed.toast.mockClear();
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
});
const input = () => container.querySelector("input")!;
async function name(value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input(), value);
    input().dispatchEvent(new Event("input", { bubbles: true }));
  });
}
function submit() {
  container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
}
async function expectPending(value: string) {
  expect(input().value).toBe(value);
  expect(input().matches(":disabled")).toBe(true);
  expect(container.querySelector('[role="status"]')?.textContent).toBe("issue.adding");
  await act(async () => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  });
  expect(input().value).toBe(value);
}

it.each(Object.values(EIssueLayoutTypes))(
  "%s keeps first request pending after a second real RHF submit completes",
  async (layout) => {
    const change = heldChange<TIssue | undefined>();
    const create = vi.fn(() => change.sent);
    await act(async () =>
      root.render(
        <QuickAddIssueRoot
          isQuickAddOpen
          layout={layout}
          quickAddCallback={create}
          prePopulatedData={{ state_id: "state", label_ids: ["label"] }}
        />
      )
    );
    await name("First");
    await act(async () => {
      submit();
      submit();
      await Promise.resolve();
    });
    expect(create).toHaveBeenCalledTimes(1);
    expect(observed.submissions).toHaveLength(2);
    await act(async () => {
      await observed.submissions[1];
    });
    expect(create.mock.calls[0]).toEqual(["A", { name: "First", state_id: "state", label_ids: ["label"] }]);
    await expectPending("First");
    expect(observed.toast).toHaveBeenCalledTimes(1);
    await act(async () => {
      change.answer(issueOf("R1"));
      await observed.submissions[0];
    });
    expect(input().value).toBe("");
    expect(input().matches(":disabled")).toBe(false);
    expect(container.querySelector('[role="status"]')).toBeNull();
  }
);

it.each(["reject", "undefined"])("%s retains input for retry", async (failure) => {
  const change = heldChange<TIssue | undefined>();
  const retry = heldChange<TIssue | undefined>();
  const create = vi.fn().mockReturnValueOnce(change.sent).mockReturnValueOnce(retry.sent);
  await act(async () =>
    root.render(<QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={create} />)
  );
  await name("First");
  await act(async () => submit());
  await act(async () => {
    if (failure === "reject") change.refuse(new Error("refused"));
    else change.answer(undefined);
    await observed.submissions[0]?.catch(() => {});
  });
  await expect(observed.toast.mock.calls[0]![0]).rejects.toBeInstanceOf(Error);
  expect(input().value).toBe("First");
  expect(input().matches(":disabled")).toBe(false);
  await act(async () => submit());
  expect(create).toHaveBeenCalledTimes(2);
  await act(async () => {
    retry.answer(issueOf("R2"));
    await observed.submissions[1];
  });
});

it.each(["resolve", "reject"])("A %s cannot reset B or release B's pending owner", async (settling) => {
  const a = heldChange<TIssue | undefined>();
  const b = heldChange<TIssue | undefined>();
  const create = vi.fn().mockReturnValueOnce(a.sent).mockReturnValueOnce(b.sent);
  const render = () =>
    root.render(<QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={create} />);
  await act(async () => render());
  await name("First");
  await act(async () => submit());
  await act(async () => {
    observed.route = { workspaceSlug: "other", projectId: "B" };
    window.dispatchEvent(new Event("routechange"));
  });
  expect(input().value).toBe("");
  await name("Second");
  await act(async () => submit());
  await act(async () => {
    if (settling === "resolve") a.answer(issueOf("A1"));
    else a.refuse(new Error("refused"));
    await observed.submissions[0]?.catch(() => {});
  });
  const originalAction = observed.toast.mock.calls[0]![1].success.actionItems(issueOf("A1"));
  expect(originalAction.props).toMatchObject({ workspaceSlug: "ws", projectId: "A", issueId: "A1" });
  await expectPending("Second");
  await act(async () => {
    submit();
    await Promise.resolve();
  });
  await act(async () => {
    await observed.submissions[2];
  });
  expect(create).toHaveBeenCalledTimes(2);
  await expectPending("Second");
  await act(async () => {
    b.answer(issueOf("B1", "B"));
    await observed.submissions[1];
  });
  expect(input().value).toBe("");
  expect(input().matches(":disabled")).toBe(false);
});

it.each(["resolve", "reject"])("unmount consumes a late %s without local effects", async (settling) => {
  const change = heldChange<TIssue | undefined>();
  await act(async () =>
    root.render(
      <QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={() => change.sent} />
    )
  );
  await name("First");
  await act(async () => submit());
  await act(async () => root.render(null));
  await act(async () => {
    if (settling === "resolve") change.answer(issueOf("R1"));
    else change.refuse(new Error("refused"));
    await observed.submissions[0];
  });
  expect(container.childElementCount).toBe(0);
});

it("keeps editable prefill but never sends entity identity or audit fields", async () => {
  const change = heldChange<TIssue | undefined>();
  const create = vi.fn(() => change.sent);
  const prefill = { ...issueOf("local-id"), tempId: "local-pending", state__group: "completed" as const };
  await act(async () =>
    root.render(
      <QuickAddIssueRoot
        isQuickAddOpen
        layout={EIssueLayoutTypes.LIST}
        prePopulatedData={prefill}
        quickAddCallback={create}
      />
    )
  );
  await name("First");
  await act(async () => submit());
  expect(create.mock.calls[0]).toEqual([
    "A",
    {
      name: "First",
      state_id: "state",
      priority: "none",
      assignee_ids: [],
      label_ids: [],
      start_date: null,
      target_date: null,
      parent_id: null,
    },
  ]);
  await act(async () => {
    change.answer(issueOf("R1"));
    await observed.submissions[0];
  });
});
