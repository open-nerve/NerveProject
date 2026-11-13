/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { observable, runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { vi } from "vitest";
import type { Module, ModuleLink, ModuleLinkCreate, ModuleLinkPatch, ModuleUpdate, Progress } from "@nerve/api-client";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import type { IModuleStore } from "@/store/planning/module.store";
import { ModuleDetailsSidebar } from "./root";

// A module's progress sidebar as a person reads and changes it (P6a spec 2.6), for its tests: the module store is a
// stand-in a test file gives the sidebar by mocking @/hooks/store/use-modules with it (sidebarModules); and where the
// sidebar renders.

/**
 * A stand-in for the module store, as the sidebar reads the module: the module, its progress and its links, which a
 * test sets (show); its changes of the module (a change of its fields, of its members) go to nerve, which settles the
 * last one sent when the test says (sent); answered, the store writes nerve's answer into the module before the change
 * resolves, as the store does. Its changes of the links (one added, changed or removed) likewise (linkSent): nerve's
 * answer goes into the module's links, a link added first, a link removed out of them.
 */
export function sidebarModules(module: Module, held: { progress?: Progress; links?: ModuleLink[] } = {}) {
  const shown = observable.box<Module | undefined>(module, { deep: false });
  const progress = observable.box(held.progress, { deep: false });
  const links = observable.box(held.links, { deep: false });
  const sent: { change: HeldChange<Module> }[] = [];
  const linkSent: { change: HeldChange<ModuleLink | undefined> }[] = [];
  const of =
    <T,>(box: { get: () => T }) =>
    (moduleId: string | null | undefined) =>
      moduleId === module.id ? box.get() : undefined;
  const changed = async () => {
    const change = heldChange<Module>();
    sent.push({ change });
    const answer = await change.sent;
    runInAction(() => shown.set(answer));
    return answer;
  };
  /** A change of the links, whose answer write gives the links that show it. */
  const linkChanged = async (write: (listed: ModuleLink[], answer: ModuleLink | undefined) => ModuleLink[]) => {
    const change = heldChange<ModuleLink | undefined>();
    linkSent.push({ change });
    const answer = await change.sent;
    runInAction(() => links.set(write(links.get() ?? [], answer)));
    return answer;
  };
  return {
    getModuleById: of(shown),
    getModuleProgress: of(progress),
    getModuleLinks: of(links),
    update: vi.fn<(moduleId: string, body: ModuleUpdate) => Promise<Module>>(changed),
    changeMembers: vi.fn<IModuleStore["changeMembers"]>(changed),
    addLink: vi.fn((_moduleId: string, _body: ModuleLinkCreate) =>
      linkChanged((listed, answer) => (answer ? [answer, ...listed] : listed))
    ),
    changeLink: vi.fn((_linkId: string, _body: ModuleLinkPatch) =>
      linkChanged((listed, answer) => listed.map((link) => (link.id === answer?.id ? answer : link)))
    ),
    removeLink: vi.fn(async (linkId: string) => {
      await linkChanged((listed) => listed.filter((link) => link.id !== linkId));
    }),
    sent,
    linkSent,
    /** The store now gives these of the module. */
    show: (fields: { module?: Module; progress?: Progress; links?: ModuleLink[] }) =>
      runInAction(() => {
        if ("module" in fields) shown.set(fields.module);
        if ("progress" in fields) progress.set(fields.progress);
        if ("links" in fields) links.set(fields.links);
      }),
  };
}

/** Where a test renders the sidebar; mount() starts it afresh, and unmount() ends it. */
export const page: { host?: HTMLDivElement; root?: ReturnType<typeof createRoot> } = {};

export function mount() {
  page.host = document.createElement("div");
  document.body.append(page.host);
  page.root = createRoot(page.host);
}

export async function unmount() {
  await act(async () => page.root?.unmount());
  page.host?.remove();
}

/** Renders the sidebar of the module of moduleId, of a project in acme; close is its handleClose. */
export const renderSidebar = (props: Omit<ComponentProps<typeof ModuleDetailsSidebar>, "workspaceSlug">) =>
  act(async () => page.root?.render(<ModuleDetailsSidebar workspaceSlug="acme" {...props} />));
