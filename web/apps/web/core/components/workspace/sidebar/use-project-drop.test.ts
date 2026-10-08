/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useProjectDrop } from "./use-project-drop";

// What a drop of a project in the caller's sidebar asks the project store (M3 design 3.18): the hook runs as a plain
// function, outside React, with a stand-in for the project store, whose moves record their arguments.

const web = projectOf("WEB", "id-acme");
const docs = projectOf("DOCS", "id-acme");
const store = vi.hoisted(() => ({
  updateProjectSortOrder: vi.fn((_project: Project, _droppedOnId: string | undefined, _atEnd: boolean) =>
    Promise.resolve()
  ),
}));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) => [web, docs].find((project) => project.id === projectId),
    updateProjectSortOrder: store.updateProjectSortOrder,
  }),
}));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
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

  it("shows a toast when the move fails", async () => {
    store.updateProjectSortOrder.mockImplementationOnce(() => Promise.reject(new Error("Project not found")));
    useProjectDrop()(docs.id, web.id, false);
    await vi.waitFor(() => expect(toasts).toHaveLength(1));
    expect(toasts).toEqual([{ type: "error", title: "error", message: "something_went_wrong" }]);
  });
});
