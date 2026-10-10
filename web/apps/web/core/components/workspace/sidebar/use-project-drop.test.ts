/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useProjectDrop } from "./use-project-drop";

// What a drop of a project in the caller's sidebar asks the project store (M3 design 3.18), and what the page says of
// it (7.1): the hook runs as a plain function, outside React, with a stand-in for the project store, whose moves record
// their arguments. Its session is fake-tab.ts's.

const web = projectOf("WEB", "id-acme");
const docs = projectOf("DOCS", "id-acme");
const store = vi.hoisted(() => ({
  updateProjectSortOrder: vi.fn(
    (_project: Project, _droppedOnId: string | undefined, _atEnd: boolean): Promise<unknown> => Promise.resolve()
  ),
}));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) => [web, docs].find((project) => project.id === projectId),
    updateProjectSortOrder: store.updateProjectSortOrder,
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  signedIn();
  store.updateProjectSortOrder.mockClear();
  toasts.length = 0;
});

describe("useProjectDrop", () => {
  it.each([false, true])("moves the project dropped, before the one it was dropped on or at the end (%s)", (atEnd) => {
    useProjectDrop()(docs.id, web.id, atEnd);
    expect(store.updateProjectSortOrder.mock.calls).toEqual([[docs, web.id, atEnd]]);
  });

  it("moves nothing for a project dropped on itself or on nothing", () => {
    const drop = useProjectDrop();
    drop(web.id, web.id, false);
    drop(web.id, undefined, true);
    drop(undefined, web.id, false);
    expect(store.updateProjectSortOrder).not.toHaveBeenCalled();
  });

  it("says why a move failed: nerve's reason, or none for one the store did not send", async () => {
    store.updateProjectSortOrder.mockImplementationOnce(() => Promise.reject(refusal(404, "project.not_found")));
    store.updateProjectSortOrder.mockImplementationOnce(() => Promise.reject(new Error("Project not found")));
    useProjectDrop()(docs.id, web.id, false);
    useProjectDrop()(web.id, docs.id, false);
    await pageSettled();
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: "errors.project_not_found" },
      { type: "error", title: "toast.error", message: "errors.unknown" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a move $settles after another tab moved this one", async ({ settle }) => {
    const move = heldChange<undefined>();
    store.updateProjectSortOrder.mockReturnValueOnce(move.sent);
    useProjectDrop()(docs.id, web.id, false);
    switchAccount();
    settle(move);
    await pageSettled();
    expect(toasts).toEqual([]);
  });
});
