/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, pick, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { WorkspaceDetails } from "./workspace-details";

// What the workspace's general page sends, and what it shows of nerve's answer (M3 design 7.1, 7.5; P8a review P12):
// the page renders on the server, for the workspace at its address, with stand-ins for its inputs, selects and button
// (fake-controls.ts) and for the time zone's select, which keep the props they were given; the test edits as an admin
// would, through their onChange, and clicks the update. Its session is fake-tab.ts's.

const page = vi.hoisted(() => {
  const zones: { onChange: (zone: string) => void }[] = [];
  return { updateWorkspace: vi.fn(), zones };
});
// Acme has no size: the form shows none, and sends none until one is picked.
const acme = workspaceOf("acme", { name: "Acme", role: 20 });
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) => (slug === acme.slug ? acme : null),
    updateWorkspace: page.updateWorkspace,
  }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("@/components/global/timezone-select", () => ({
  TimezoneSelect: (props: { onChange: (zone: string) => void }) => {
    page.zones.push(props);
    return null;
  },
}));
vi.mock("@/components/workspace/delete-workspace-section", () => ({ DeleteWorkspaceSection: () => null }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the page, has edit change its fields, and clicks the update; settles once the page has followed nerve. */
async function update(edit: (form: { name: string; size?: string; zone?: string }) => void = () => {}) {
  renderToStaticMarkup(<WorkspaceDetails />);
  const [name] = shown.inputs;
  const [size] = shown.selects;
  const [zone] = page.zones;
  const button = shown.buttons.at(-1);
  if (!name || !size || !zone || !button?.onClick) throw new Error("the page showed no form to update");
  const form: { name: string; size?: string; zone?: string } = { name: acme.name };
  edit(form);
  name.onChange({ target: { value: form.name } });
  if (form.size) pick(size, form.size);
  if (form.zone) zone.onChange(form.zone);
  button.onClick({ preventDefault: () => {} });
  await pageSettled();
}

beforeEach(() => {
  signedIn();
  page.updateWorkspace.mockReset();
  page.updateWorkspace.mockResolvedValue(acme);
  page.zones.length = 0;
  toasts.length = 0;
  emptyShown();
});

describe("WorkspaceDetails", () => {
  it("sends the fields its form edits, the size as nerve names it, and says the workspace is updated", async () => {
    await update((form) => Object.assign(form, { name: "Acme Corp", size: "11-50", zone: "Asia/Shanghai" }));
    expect(page.updateWorkspace.mock.calls).toEqual([
      ["acme", { name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai" }],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Workspace updated successfully" }]);
  });

  // nerve cannot set the size to none (WorkspaceUpdate): a null would be refused
  it("sends no size while the workspace has none and none was picked", async () => {
    await update();
    expect(page.updateWorkspace.mock.calls).toEqual([["acme", { name: "Acme", timezone: "UTC" }]]);
  });

  it("shows nerve's reason when it refuses", async () => {
    page.updateWorkspace.mockRejectedValueOnce(refusal(403, "forbidden"));
    await update();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "says nothing when the update $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      page.updateWorkspace.mockReturnValueOnce(change.sent);
      await update();
      expect(page.updateWorkspace).toHaveBeenCalledTimes(1);
      switchAccount();
      settle(change);
      await pageSettled();
      expect(toasts).toEqual([]);
    }
  );
});
