/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Dispatch, SetStateAction } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { ArchiveRestoreProjectModal } from "./archive-restore-modal";

// What archiving and restoring a project send and do on the page (M3 design 7.1): the modal renders on the server,
// with stand-ins for its buttons (fake-controls.ts), which keep the props they were given; the test presses its
// second, as a person would. Its session is fake-tab.ts's. A render on the server shows no later state: React's
// useState, as the test gives it, keeps each value the modal gives its one state, busy, as it gives it.

const web = projectOf("WEB", "w-acme", { name: "Web" });
const store = vi.hoisted(() => ({ archiveProject: vi.fn(), restoreProject: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
const busy = vi.hoisted((): unknown[] => []);
vi.mock("react", async (importOriginal) => {
  const react = await importOriginal<typeof import("react")>();
  function useState<S>(initial: S): [S, Dispatch<SetStateAction<S>>] {
    const [state, setState] = react.useState(initial);
    return [
      state,
      (next) => {
        busy.push(next);
        setState(next);
      },
    ];
  }
  return { ...react, useState };
});
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: () => web,
    archiveProject: store.archiveProject,
    restoreProject: store.restoreProject,
  }),
}));
vi.mock("react-router", () => ({ useNavigate: () => navigate }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the modal of Web, to archive it or to restore it, and presses Archive or Restore; gives onClose. */
function press(archive: boolean) {
  const onClose = vi.fn();
  renderToStaticMarkup(
    <ArchiveRestoreProjectModal workspaceSlug="acme" projectId={web.id} isOpen onClose={onClose} archive={archive} />
  );
  shown.buttons[1]?.onClick?.({ preventDefault: () => {} });
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.archiveProject.mockReset();
  store.archiveProject.mockResolvedValue(web);
  store.restoreProject.mockReset();
  store.restoreProject.mockResolvedValue(web);
  navigate.mockReset();
  toasts.length = 0;
  busy.length = 0;
  emptyShown();
});

describe("ArchiveRestoreProjectModal", () => {
  it.each([
    {
      change: "archives",
      archive: true,
      sent: () => store.archiveProject,
      said: { title: "Archive success", message: "Web has been archived successfully" },
    },
    {
      change: "restores",
      archive: false,
      sent: () => store.restoreProject,
      said: { title: "Restore success", message: "You can find Web in your projects." },
    },
  ])("$change the project, says so and gives way to the projects", async ({ archive, sent, said }) => {
    const onClose = press(archive);
    await pageSettled();
    expect(sent().mock.calls).toEqual([[web.id]]);
    expect(toasts).toEqual([{ type: "success", ...said }]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/acme/projects"]]]);
  });

  it("stays busy until it has given way to the projects", async () => {
    const landing = heldChange<undefined>();
    navigate.mockReturnValueOnce(landing.sent);
    press(true);
    await pageSettled();
    expect([navigate.mock.calls.length, busy]).toEqual([1, [true]]);
    landing.answer(undefined);
    await pageSettled();
    expect(busy).toEqual([true, false]);
  });

  it("shows nerve's reason when it refuses, and stays", async () => {
    store.archiveProject.mockRejectedValueOnce(refusal(403, "forbidden"));
    const onClose = press(true);
    await pageSettled();
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlingsAnswering(web))(
    "neither moves nor speaks when the archiving $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const archiving = heldChange<Project>();
      store.archiveProject.mockReturnValueOnce(archiving.sent);
      const onClose = press(true);
      switchAccount();
      settle(archiving);
      await pageSettled();
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
