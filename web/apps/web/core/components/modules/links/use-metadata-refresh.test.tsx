// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { observable, runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ModuleLink } from "@nerve/api-client";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { useSessionSWR } from "@/lib/use-session-swr";
import { moduleLinkOf } from "@/store/planning/fake-modules";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { ModuleLinksList } from "./list";

// The bounded reads of a module's links while nerve reads their pages (P6a spec 2.8, R10): nerve answers a link added,
// or one whose address changed, with its metadata empty, and reads its page in the background. While a link shown has
// empty metadata, the list reads the module's links again once a second, through the SWR read of the module's links its
// page makes (MODULE_LINKS), one read at a time; it stops once every link has metadata (a failed read's carries its
// error), after thirty seconds (thirty reads at most), once nerve refuses a read, and as the list goes. It reads
// nothing while no link wants metadata. A link added, or an address changed, starts it again. The page is a stand-in
// that makes the read of the module's links; the module store a stand-in whose reads nerve answers when the test says,
// or at once, or a while after.

type StandIn = ReturnType<typeof standIn>;
const held = vi.hoisted(() => ({ modules: undefined as unknown as StandIn, loginId: "x" }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: held.loginId }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));

const auth = moduleOf(projectOf("WEB", "id-acme"), "Auth");
/** A link whose page nerve has not read yet, and one it has. */
const docs = moduleLinkOf(auth, "https://docs.example.com");
const blog = moduleLinkOf(auth, "https://blog.example.com", { metadata: { title: "Blog" } });
const DOCS_READ: ModuleLink = { ...docs, metadata: { title: "Docs", favicon: "data:image/png;base64,AA==" } };

/** The module store's links, and its reads of them: each nerve answers when the test says, or as answering says. */
function standIn() {
  const shown = observable.box<ModuleLink[] | undefined>(undefined, { deep: false });
  const reads: { nerve: HeldChange<ModuleLink[]>; at: number }[] = [];
  const answers: { links?: ModuleLink[]; after: number } = { after: 0 };
  return {
    getModuleLinks: (id: string | null | undefined) => (id === auth.id ? shown.get() : undefined),
    fetchLinks: vi.fn(async (_id: string) => {
      const nerve = heldChange<ModuleLink[]>();
      reads.push({ nerve, at: Date.now() });
      const { links, after } = answers;
      if (links) setTimeout(() => nerve.answer(links), after);
      const answer = await nerve.sent;
      runInAction(() => shown.set(answer));
      return answer;
    }),
    reads,
    /** nerve answers each read from now on with links, after so many milliseconds. */
    answering: (links: ModuleLink[], after = 0) => Object.assign(answers, { links, after }),
    /** The store now holds links (a link added, say). */
    hold: (links: ModuleLink[]) => runInAction(() => shown.set(links)),
  };
}

/** The module's page: it reads the module's links, and shows them while listed. */
function Page({ listed }: { listed: boolean }) {
  const modules = held.modules;
  useSessionSWR(["MODULE_LINKS", auth.id], (id) => modules.fetchLinks(id));
  return listed ? (
    <ModuleLinksList
      moduleId={auth.id}
      editable={false}
      removing={new Set()}
      onEdit={() => undefined}
      onRemove={() => undefined}
    />
  ) : null;
}

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 13, 12) });
  held.loginId = "x";
  held.modules = standIn();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});

const tick = (ms: number) => act(async () => vi.advanceTimersByTimeAsync(ms));
const render = (listed = true) =>
  act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <Page listed={listed} />
      </SWRConfig>
    )
  );
/** The reads the list made, after its page's own. */
const reads = () => held.modules.reads.slice(1);
/** nerve answers, or refuses, the list's read of index. */
const answer = (index: number, links: ModuleLink[]) => act(async () => reads()[index]?.nerve.answer(links));
const refuse = (index: number, error: unknown) => act(async () => reads()[index]?.nerve.refuse(error));
const anchors = () => [...host.querySelectorAll("a")].map((a) => a.textContent);

/** The page shows the module's links as nerve gives them to its read; the time then. */
async function opened(links: ModuleLink[]) {
  await render();
  await act(async () => held.modules.reads[0]?.nerve.answer(links));
  return Date.now();
}

it("reads the links again once a second while one has no metadata, one read at a time, until each has", async () => {
  await opened([docs, blog]);
  expect(anchors()).toEqual(["https://docs.example.com", "Blog"]);
  await tick(999);
  expect(reads()).toHaveLength(0);
  await tick(1);
  expect(reads()).toHaveLength(1);
  // nerve is slow: no second read while the first is out
  await tick(5000);
  expect(reads()).toHaveLength(1);
  await answer(0, [docs, blog]);
  await tick(999);
  expect(reads()).toHaveLength(1);
  await tick(1);
  expect(reads()).toHaveLength(2);
  await answer(1, [DOCS_READ, blog]);
  expect(anchors()).toEqual(["Docs", "Blog"]);
  expect(host.querySelector("img")?.getAttribute("src")).toBe(DOCS_READ.metadata.favicon);
  await tick(31000);
  expect(reads()).toHaveLength(2);
});

it.each([
  ["no links", []],
  ["links each with its page's metadata", [blog, { ...docs, metadata: { title: null } }]],
  ["a link whose page nerve could not read", [{ ...docs, metadata: { error: "failed" as const } }]],
  ["a link whose page nerve would not read", [{ ...docs, metadata: { error: "blocked" as const } }]],
])("reads nothing again while the module has %s", async (_, links) => {
  await opened(links);
  await tick(31000);
  expect(reads()).toHaveLength(0);
});

it("reads at most thirty times, none thirty seconds after it began, though nerve keeps giving no metadata", async () => {
  held.modules.answering([docs]);
  const began = await opened([docs]);
  await tick(60000);
  expect(reads().length).toBeGreaterThan(0);
  expect(reads().length).toBeLessThanOrEqual(30);
  expect(reads().every(({ at }) => at - began < 30000)).toBe(true);
  // the address still shows, its icon by the address
  expect(anchors()).toEqual(["https://docs.example.com"]);
  expect(host.querySelector("img")).toBeNull();
});

it("begins no read thirty seconds after it began, though nerve is slow to answer", async () => {
  held.modules.answering([docs], 2500);
  const began = await opened([docs]);
  await tick(120000);
  expect(reads().length).toBeGreaterThan(5);
  expect(reads().every(({ at }) => at - began < 30000)).toBe(true);
});

it("begins again for a link added after it stopped, never while a read is out", async () => {
  held.modules.answering([docs]);
  await opened([docs]);
  await tick(31000);
  const count = reads().length;
  await tick(10000);
  expect(reads()).toHaveLength(count);
  const notes = moduleLinkOf(auth, "https://notes.example.com");
  held.modules.answering([notes, docs], 3000);
  held.modules.hold([notes, docs]);
  await tick(1000);
  expect(reads()).toHaveLength(count + 1);
  // the address of the new link changes while that read is out: the next read waits for it
  held.modules.hold([{ ...notes, url: "https://notes.example.org" }, docs]);
  await tick(1500);
  expect(reads()).toHaveLength(count + 1);
});

it.each([
  ["before its next read", 500],
  ["while a read is out", 1000],
])("stops as the list goes %s, and follows nothing of a read still out", async (_, after) => {
  await opened([docs]);
  await tick(after);
  const out = reads().length;
  await render(false);
  await answer(0, [docs]);
  await tick(31000);
  expect(reads()).toHaveLength(out);
});

it("stops once nerve refuses a read", async () => {
  await opened([docs]);
  await tick(1000);
  await refuse(0, refusal(503, "server_busy"));
  await tick(31000);
  expect(reads()).toHaveLength(1);
});

it("stops as the session ends: the next session's links are its own", async () => {
  await opened([docs]);
  await tick(1000);
  const before = held.modules;
  held.loginId = "y";
  held.modules = standIn();
  await render();
  await act(async () => held.modules.reads[0]?.nerve.answer([DOCS_READ]));
  await act(async () => before.reads[1]?.nerve.answer([docs]));
  await tick(31000);
  expect(before.reads).toHaveLength(2);
  expect(held.modules.reads).toHaveLength(1);
});
