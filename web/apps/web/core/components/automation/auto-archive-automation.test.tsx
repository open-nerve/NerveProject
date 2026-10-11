/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ComponentProps } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { AutoArchiveAutomation } from "./auto-archive-automation";
import type { SelectMonthModal } from "./select-month-modal";

// What the auto-archiving's controls ask the project store (v0 design 7.7: a change whose body depends on what the
// store holds is built in its turn), and what the page does once nerve has answered (M3 design 7.1): the component
// renders on the server for web, whose closed work items are archived after 3 months, with stand-ins for its switch
// and its select of months (fake-controls.ts) and for its custom range, which keep the props they were given, and for
// the project store, whose changes record their arguments. Its session is fake-tab.ts's.

const web: Project = projectOf("WEB", "w-acme", { archive_in: 3 });
const store = vi.hoisted(() => ({
  updateProject: vi.fn((_projectId: string, _data: ProjectUpdate): Promise<unknown> => Promise.resolve()),
  toggleAutoArchive: vi.fn((_projectId: string): Promise<unknown> => Promise.resolve()),
}));
/** The props the custom range was given, in the order rendered. */
const ranges = vi.hoisted((): ComponentProps<typeof SelectMonthModal>[] => []);
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) => (projectId === web.id ? web : undefined),
    updateProject: store.updateProject,
    toggleAutoArchive: store.toggleAutoArchive,
  }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme", projectId: web.id }) }));
vi.mock("@/components/automation", () => ({
  SelectMonthModal: (props: ComponentProps<typeof SelectMonthModal>) => {
    ranges.push(props);
    return null;
  },
}));
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** Renders web's auto-archiving: gives its switch, its select of months and its custom range. */
function render() {
  emptyShown();
  ranges.length = 0;
  renderToStaticMarkup(<AutoArchiveAutomation />);
  const [toggle] = shown.switches;
  const [months] = shown.selects;
  const [range] = ranges;
  if (!toggle || !months || !range) throw new Error("the auto-archiving showed no switch, select or custom range");
  return { toggle, months, range };
}

beforeEach(() => {
  signedIn();
  store.updateProject.mockClear();
  store.toggleAutoArchive.mockClear();
  toasts.length = 0;
});

describe("AutoArchiveAutomation", () => {
  it("turns the auto-archiving with the store's toggle, from nerve's last answer, not from what the switch shows", async () => {
    const { toggle } = render();
    await toggle.onCheckedChange(false);
    expect(store.toggleAutoArchive.mock.calls).toEqual([[web.id]]);
    expect(store.updateProject).not.toHaveBeenCalled();
  });

  it("sends the months picked", () => {
    const { months } = render();
    months.onChange(6);
    expect(store.updateProject.mock.calls).toEqual([[web.id, { archive_in: 6 }]]);
    expect(store.toggleAutoArchive).not.toHaveBeenCalled();
  });

  it("shows nerve's reason in a toast when it refuses the turn", async () => {
    store.toggleAutoArchive.mockImplementationOnce(() => Promise.reject(refusal(403, "forbidden")));
    const { toggle } = render();
    await toggle.onCheckedChange(false);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)("says nothing when the turn $settles after another tab moved this one", async ({ settle }) => {
    const turn = heldChange<undefined>();
    store.toggleAutoArchive.mockReturnValueOnce(turn.sent);
    const { toggle } = render();
    const turned = toggle.onCheckedChange(false);
    switchAccount();
    settle(turn);
    await turned;
    expect(toasts).toEqual([]);
  });

  it("closes the custom range once nerve has made the months it sends", async () => {
    const { range } = render();
    const made = vi.fn();
    await range.handleChange({ archive_in: 6 }, made);
    expect(store.updateProject.mock.calls).toEqual([[web.id, { archive_in: 6 }]]);
    expect(made).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([]);
  });

  it("keeps the custom range open when nerve refuses its months, and shows nerve's reason in a toast", async () => {
    store.updateProject.mockImplementationOnce(() => Promise.reject(refusal(403, "forbidden")));
    const { range } = render();
    const made = vi.fn();
    await range.handleChange({ archive_in: 6 }, made);
    expect(made).not.toHaveBeenCalled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "neither closes the custom range nor says anything when its change $settles after another tab moved this one",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      store.updateProject.mockReturnValueOnce(change.sent);
      const { range } = render();
      const made = vi.fn();
      const sent = range.handleChange({ archive_in: 6 }, made);
      switchAccount();
      settle(change);
      await sent;
      expect(made).not.toHaveBeenCalled();
      expect(toasts).toEqual([]);
    }
  );
});
