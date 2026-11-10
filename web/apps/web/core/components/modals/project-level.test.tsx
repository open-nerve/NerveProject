// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { CommandPaletteStore } from "@/store/command-palette.store";
import { ProjectLevelModals } from "./project-level";

// The project's dialogs' host (P5a spec 2.6): their open flags are the command palette store's, which outlives the
// project, so the host closes them as it goes. Each dialog is a stand-in that shows whether it is open, and where.
const palette = vi.hoisted(() => ({ store: undefined as CommandPaletteStore | undefined }));
type Dialog = { isOpen: boolean; projectId: string };
const shown = ({ isOpen, projectId }: Dialog, what: string) => (isOpen ? <p>{`${what} of ${projectId}`}</p> : null);
vi.mock("@/lib/store-context", () => ({ rootStore: { powerK: {} } }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "host" } }, publicClient: {} }));
vi.mock("@/hooks/store/use-command-palette", () => ({ useCommandPalette: () => palette.store }));
vi.mock("@/components/cycles/modal", () => ({ CycleCreateUpdateModal: (props: Dialog) => shown(props, "cycle") }));
vi.mock("@/components/modules", () => ({ CreateUpdateModuleModal: (props: Dialog) => shown(props, "module") }));
vi.mock("@/components/views/modal", () => ({ CreateUpdateProjectViewModal: (props: Dialog) => shown(props, "view") }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  palette.store = new CommandPaletteStore();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
const render = (projectId: string | null) =>
  act(async () =>
    root.render(projectId && <ProjectLevelModals key={projectId} workspaceSlug="ws" projectId={projectId} />)
  );
const openAll = () =>
  act(async () => {
    palette.store!.toggleCreateCycleModal(true);
    palette.store!.toggleCreateModuleModal(true);
    palette.store!.toggleCreateViewModal(true);
  });
const flags = () => [
  palette.store!.isCreateCycleModalOpen,
  palette.store!.isCreateModuleModalOpen,
  palette.store!.isCreateViewModalOpen,
];

it("keeps its dialogs open while its project stays", async () => {
  await render("A");
  await openAll();
  await render("A");
  expect(host.textContent).toBe("cycle of Amodule of Aview of A");
  expect(flags()).toEqual([true, true, true]);
});

it("closes the three dialogs it renders as another project's host replaces it, which opens none of them", async () => {
  await render("A");
  await openAll();
  await render("B");
  expect(flags()).toEqual([false, false, false]);
  expect(host.textContent).toBe("");
});

it("closes them as it goes with no project's host after it", async () => {
  await render("A");
  await openAll();
  await render(null);
  expect(flags()).toEqual([false, false, false]);
  // a Back to the project opens none of them again
  await render("A");
  expect(host.textContent).toBe("");
});
