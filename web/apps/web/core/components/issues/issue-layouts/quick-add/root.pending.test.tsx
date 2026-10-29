// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import type { Issue, IssueCreate } from "@nerve/api-client";
import { act } from "react";
import { createRoot } from "react-dom/client";
import type { FieldValues, UseFormProps } from "react-hook-form";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { EIssueLayoutTypes } from "@nerve/types";
import { FakeNerve, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { IssueApiService } from "@/services/issue/issue-api.service";
import { projectOf } from "@/store/project/fake-projects";
import { issueOf, issueDetailOf } from "@/store/issue/fake-issues";
import { QuickAddIssueRoot } from "./root";

const observed = vi.hoisted(() => ({
  route: { workspaceSlug: "ws", projectId: "A" },
  submissions: [] as Promise<unknown>[],
  settledToast: vi.fn(),
  session: { loginId: "session-A" },
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
vi.mock("@nerve/propel/toast", () => ({
  TOAST_TYPE: { SUCCESS: "success", ERROR: "error" },
  setToast: observed.settledToast,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: observed.session } }));
vi.mock("../../create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
const projectA = projectOf("A", "workspace", { id: "A" });
const projectB = projectOf("B", "other-workspace", { id: "B" });
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (id: string) => (id === projectA.id ? projectA : id === projectB.id ? projectB : undefined),
  }),
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let nerve: FakeNerve;
let api: IssueApiService;
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(() => {
  vi.useFakeTimers();
  nerve = new FakeNerve();
  api = new IssueApiService(nerve.client());
  observed.route = { workspaceSlug: "ws", projectId: "A" };
  observed.submissions.length = 0;
  observed.settledToast.mockClear();
  observed.session.loginId = "session-A";
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  vi.useRealTimers();
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

const createHTTP = () =>
  vi.fn(
    (project: string | null | undefined, body: IssueCreate): Promise<Issue | undefined> =>
      project ? api.create(project, body) : Promise.resolve(undefined)
  );
async function requestCount(count: number) {
  await act(async () => until(() => nerve.calls.length === count, "quick-add HTTP"));
}
async function answer(index: number, id: string, project = "A") {
  await act(async () => {
    nerve.calls[index]!.answer(
      json(201, {
        ...issueDetailOf(id, project),
        workspace_id: project === "A" ? projectA.workspace_id : projectB.workspace_id,
      })
    );
    await observed.submissions[index];
  });
}

it.each(["resolve", "reject"])("retired session ignores late quick-add %s followers", async (settling) => {
  await act(async () =>
    root.render(<QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={createHTTP()} />)
  );
  await name("First");
  await act(async () => submit());
  await requestCount(1);
  observed.session.loginId = "session-B";
  await act(async () => {
    nerve.calls[0]!.answer(settling === "resolve" ? json(201, issueDetailOf("R1")) : json(403, { code: "forbidden" }));
    await observed.submissions[0];
  });
  expect(observed.settledToast).not.toHaveBeenCalled();
  expect(input().value).toBe("First");
});

it.each(Object.values(EIssueLayoutTypes))(
  "%s preserves first request owner after duplicate real RHF submit",
  async (layout) => {
    const create = createHTTP();
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
    await requestCount(1);
    await act(async () => {
      await observed.submissions[1];
    });
    expect(nerve.calls[0]).toMatchObject({
      method: "POST",
      path: "/api/v0/projects/A/issues",
      body: { name: "First", state_id: "state", label_ids: ["label"] },
    });
    expect(create).toHaveBeenCalledTimes(1);
    expect(observed.submissions).toHaveLength(2);
    await expectPending("First");
    await answer(0, "R1");
    expect(input().value).toBe("");
    expect(input().matches(":disabled")).toBe(false);
    expect(container.querySelector('[role="status"]')).toBeNull();
  }
);

it.each(["reject", "undefined"])("%s retains input for retry", async (failure) => {
  const create = createHTTP();
  if (failure === "undefined") create.mockImplementationOnce(() => Promise.resolve(undefined));
  await act(async () =>
    root.render(<QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={create} />)
  );
  await name("First");
  await act(async () => submit());
  if (failure === "reject") {
    await requestCount(1);
    await act(async () => {
      nerve.calls[0]!.answer(json(403, { code: "forbidden" }));
      await observed.submissions[0];
    });
  } else
    await act(async () => {
      await observed.submissions[0];
    });
  expect(input().value).toBe("First");
  expect(input().matches(":disabled")).toBe(false);
  await act(async () => submit());
  await requestCount(failure === "reject" ? 2 : 1);
  await act(async () => {
    nerve.calls.at(-1)!.answer(json(201, issueDetailOf("retry")));
    await observed.submissions[1];
  });
  expect(input().value).toBe("");
});

it.each(["resolve", "reject"])("A %s cannot reset B or release B's pending owner", async (settling) => {
  const create = createHTTP();
  await act(async () =>
    root.render(<QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={create} />)
  );
  await name("First");
  await act(async () => submit());
  await requestCount(1);
  await act(async () => {
    observed.route = { workspaceSlug: "other", projectId: "B" };
    window.dispatchEvent(new Event("routechange"));
  });
  expect(input().value).toBe("");
  await name("Second");
  await act(async () => submit());
  await requestCount(2);
  expect(nerve.calls.map((call) => call.path)).toEqual(["/api/v0/projects/A/issues", "/api/v0/projects/B/issues"]);
  await act(async () => {
    nerve.calls[0]!.answer(settling === "resolve" ? json(201, issueDetailOf("A1")) : json(403, { code: "forbidden" }));
    await observed.submissions[0];
  });
  if (settling === "resolve")
    expect(observed.settledToast.mock.calls[0]![0].actionItems.props).toMatchObject({
      workspaceSlug: "ws",
      projectId: "A",
      issueId: "A1",
    });
  await expectPending("Second");
  await act(async () => {
    submit();
    await Promise.resolve();
  });
  await act(async () => {
    await observed.submissions[2];
  });
  expect(nerve.calls).toHaveLength(2);
  await expectPending("Second");
  await answer(1, "B1", "B");
  expect(input().value).toBe("");
});

it.each(["resolve", "reject"])("unmount consumes late HTTP %s without local effects", async (settling) => {
  await act(async () =>
    root.render(<QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={createHTTP()} />)
  );
  await name("First");
  await act(async () => submit());
  await requestCount(1);
  await act(async () => root.render(null));
  await act(async () => {
    nerve.calls[0]!.answer(settling === "resolve" ? json(201, issueDetailOf("R1")) : json(403, { code: "forbidden" }));
    await observed.submissions[0];
  });
  expect(container.childElementCount).toBe(0);
});

it("editablePrefillEmitsOnlyIssueCreateFieldsAndOmitsNullDatesAndParent", async () => {
  await act(async () =>
    root.render(
      <QuickAddIssueRoot
        isQuickAddOpen
        layout={EIssueLayoutTypes.LIST}
        prePopulatedData={{ ...issueOf("local-id", projectA.id, projectA.workspace_id) }}
        quickAddCallback={createHTTP()}
      />
    )
  );
  await name("First");
  await act(async () => submit());
  await requestCount(1);
  expect(nerve.calls[0]!.body).toEqual({
    name: "First",
    state_id: "state",
    priority: "none",
    assignee_ids: [],
    label_ids: [],
  });
  await answer(0, "R1");
});
