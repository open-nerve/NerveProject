/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useJoinProject } from "./use-join-project";

// Joining a project from its card or its own page (M3 design 3.5, 7.1, 7.6): the hook runs as a plain function, with
// a stand-in for the store's join, which nerve settles when the test says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ joinProject: vi.fn(), done: vi.fn() }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ joinProject: page.joinProject }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme", { member_role: 15 });

beforeEach(() => {
  signedIn();
  page.joinProject.mockReset();
  page.joinProject.mockResolvedValue(web);
  page.done.mockReset();
  toasts.length = 0;
});

describe("useJoinProject", () => {
  it("joins the project, then does what the page does next", async () => {
    await useJoinProject()(web.id, page.done);
    expect([page.joinProject.mock.calls, page.done.mock.calls, toasts]).toEqual([[[web.id]], [[web]], []]);
  });

  it("joins the project with nothing next where the store's answer is the page's", async () => {
    await useJoinProject()(web.id);
    expect([page.joinProject.mock.calls, toasts]).toEqual([[[web.id]], []]);
  });

  it("shows nerve's reason when it refuses, and does nothing next", async () => {
    page.joinProject.mockRejectedValueOnce(refusal(404, "project.not_found"));
    await useJoinProject()(web.id, page.done);
    expect([page.done.mock.calls, toasts]).toEqual([
      [],
      [{ type: "error", title: "toast.error", message: "errors.project_not_found" }],
    ]);
  });

  it.each(lateSettlings)(
    "does nothing on the page when the join $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const answer = heldChange<undefined>();
      page.joinProject.mockReturnValueOnce(answer.sent);
      const joined = useJoinProject()(web.id, page.done);
      switchAccount();
      settle(answer);
      await joined;
      expect([page.done.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
