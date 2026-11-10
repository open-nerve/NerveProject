// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { usePowerKCycleContextBasedActions } from "./commands";

// The command palette's commands on a cycle's page (P5a spec 2.7, R2): its link's copy alone, through the web app's
// one copy (use-copy-link.ts), of the route's cycle; the cycles' favorites are gone until M7 brings them back (M6 design
// 15 #13). The clipboard's write is a stand-in the test lets or refuses.

const held = vi.hoisted(() => ({
  params: { workspaceSlug: "acme", projectId: "p-web", cycleId: "c-sprint" },
  copy: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("react-router", () => ({ useParams: () => held.params }));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyUrlToClipboard: held.copy,
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>, commands: TPowerKCommandConfig[];
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const ctx = {
  params: held.params,
  activeCommand: null,
  activeContext: "cycle",
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: () => {},
} satisfies TPowerKContext;
function Commands() {
  commands = usePowerKCycleContextBasedActions();
  return null;
}
beforeEach(async () => {
  signedIn();
  toasts.length = 0;
  held.copy.mockReset().mockResolvedValue(undefined);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () => root.render(<Commands />));
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});

/** The palette runs the command of id, as it does once chosen. */
const run = (id: string) =>
  act(async () => {
    const command = commands.find((one) => one.id === id);
    if (command?.type !== "action") throw new Error(`no action ${id}`);
    await command.action(ctx);
  });

it("gives a cycle's page its link's copy alone, to anyone who sees the page: no favorite", () => {
  expect(commands.map((command) => [command.id, command.isVisible(ctx), command.isEnabled(ctx)])).toEqual([
    ["copy_cycle_url", true, true],
  ]);
});

it("copies the link of the route's cycle, saying so", async () => {
  await run("copy_cycle_url");
  expect(held.copy.mock.calls).toEqual([["/acme/projects/p-web/cycles/c-sprint"]]);
  expect(toasts).toEqual([
    {
      type: "success",
      title: "common.link_copied",
      message: "power_k.contextual_actions.cycle.copy_url_toast_success",
    },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  await run("copy_cycle_url");
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});
