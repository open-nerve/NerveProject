/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { DeleteProjectModal } from "./delete-project-modal";

// What the deletion of a project sends and does on the page (M3 design 7.1, 9.5): the modal renders on the server,
// with stand-ins for its inputs and buttons (fake-controls.ts), which keep the props they were given; the test types
// as a person would, through the inputs' onChange, and submits the form. Its session is fake-tab.ts's; the address's
// params are params'.

const store = vi.hoisted(() => ({ deleteProject: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
const params = vi.hoisted((): { workspaceSlug?: string; projectId?: string } => ({}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ deleteProject: store.deleteProject }) }));
vi.mock("react-router", () => ({ useNavigate: () => navigate, useParams: () => params }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme", { name: "Web" });

/** Renders the modal of Web, types name and words into its two inputs, and submits it; gives onClose. */
async function confirm(name: string, words: string) {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeleteProjectModal isOpen project={web} onClose={onClose} />);
  const [nameInput, wordsInput] = shown.inputs;
  nameInput?.onChange({ target: { value: name } });
  wordsInput?.onChange({ target: { value: words } });
  await submitModalForm();
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.deleteProject.mockReset();
  store.deleteProject.mockResolvedValue(undefined);
  navigate.mockClear();
  Object.assign(params, { workspaceSlug: "acme", projectId: web.id });
  toasts.length = 0;
  emptyShown();
});

const deleted = { type: "success", title: "Success!", message: "Project deleted successfully." };

describe("DeleteProjectModal", () => {
  it("deletes the project once its name and the words are typed; its own page gives way to the projects", async () => {
    const onClose = await confirm("Web", "delete my project");
    expect(store.deleteProject.mock.calls).toEqual([[web]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/acme/projects"]]]);
    expect(toasts).toEqual([deleted]);
  });

  it.each([
    { page: "the archived projects'", projectId: undefined },
    { page: "another project's", projectId: "p-ops" },
  ])("stays on a page that is not the project's, $page, and says so", async ({ projectId }) => {
    params.projectId = projectId;
    const onClose = await confirm("Web", "delete my project");
    expect(store.deleteProject.mock.calls).toEqual([[web]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, []]);
    expect(toasts).toEqual([deleted]);
  });

  it.each([
    { typed: "another name", name: "Web app", words: "delete my project" },
    { typed: "other words", name: "Web", words: "delete it" },
  ])("deletes nothing when the form has $typed", async ({ name, words }) => {
    const onClose = await confirm(name, words);
    expect([store.deleteProject.mock.calls, onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], [], []]);
  });

  it("shows nerve's reason when it refuses, and stays", async () => {
    store.deleteProject.mockRejectedValueOnce(refusal(403, "forbidden"));
    const onClose = await confirm("Web", "delete my project");
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "neither moves nor speaks when the deletion $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const deletion = heldChange<undefined>();
      store.deleteProject.mockReturnValueOnce(deletion.sent);
      const submitted = confirm("Web", "delete my project");
      await vi.waitFor(() => expect(store.deleteProject).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(deletion);
      const onClose = await submitted;
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
