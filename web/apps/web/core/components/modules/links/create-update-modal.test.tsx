// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ModuleLink } from "@nerve/api-client";
import { heldChange, signedIn, switchAccount, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleLinkOf } from "@/store/planning/fake-modules";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { ModuleLinkModal } from "./create-update-modal";

// The dialog that adds a link to a module, or changes one of its links (P6a spec 2.8, R11), as a person uses it: its
// address, required, goes as typed (nerve trims it and gives it a scheme it lacks); its title, which may be left
// empty, as none. It closes only once nerve has added or changed the link, and says so; refused, it stays with what it
// holds, nerve's reason under the field it names (422) or in a toast (a link the module has already, an archived
// module, a role that may not, nerve busy). The module store is a stand-in whose changes nerve settles when the test
// says (fake-tab.ts).

const ui = vi.hoisted(() => ({ addLink: vi.fn(), changeLink: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({
  useModules: () => ({ addLink: ui.addLink, changeLink: ui.changeLink }),
}));

const auth = moduleOf(projectOf("WEB", "id-acme"), "Auth");
const docs = moduleLinkOf(auth, "https://docs.example.com", { title: "Docs", metadata: { title: "Docs" } });
const close = vi.fn();
let sent: HeldChange<ModuleLink>;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 13, 12) });
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  signedIn();
  toasts.length = 0;
  close.mockReset();
  sent = heldChange<ModuleLink>();
  ui.addLink.mockReset().mockImplementation(() => sent.sent);
  ui.changeLink.mockReset().mockImplementation(() => sent.sent);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

/** A sidebar with the control that opens the dialog: a new link's, or the change of link. */
function Sidebar({ link }: { link?: ModuleLink }) {
  const [shown, setShown] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setShown(true)}>
        opener
      </button>
      <ModuleLinkModal
        isOpen={shown}
        handleClose={() => {
          close();
          setShown(false);
        }}
        moduleId={auth.id}
        link={link}
      />
    </>
  );
}
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const opener = () => [...host.querySelectorAll("button")].find((button) => button.textContent === "opener")!;
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');
const input = (id: "url" | "title") => document.querySelector<HTMLInputElement>(`#${id}`)!;
const button = (name: string) =>
  [...(dialog()?.querySelectorAll("button") ?? [])].find((item) => item.textContent?.trim() === name)!;
const title = () => document.getElementById(dialog()!.getAttribute("aria-labelledby") ?? "")?.textContent;
const described = (id: "url" | "title") =>
  (input(id).getAttribute("aria-describedby")?.split(/\s+/) ?? [])
    .map((one) => document.getElementById(one)?.textContent ?? "")
    .join(" ");
async function open(link?: ModuleLink) {
  await act(async () => root.render(<Sidebar link={link} />));
  await act(async () => {
    opener().focus();
    opener().click();
  });
  await settle();
  expect(dialog()).not.toBeNull();
}
async function fill(id: "url" | "title", value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input(id), value);
    input(id).dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function submit() {
  await act(async () =>
    dialog()!
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await settle();
}
const answer = async (link: ModuleLink) => {
  await act(async () => sent.answer(link));
  await settle();
};
const refuse = async (error: unknown) => {
  await act(async () => sent.refuse(error));
  await settle();
};
const escape = async () => {
  await act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await settle();
};

it("adds a link with its address as typed and no title; closing and saying so once nerve has added it", async () => {
  await open();
  expect(title()).toBe("common.add_link");
  await fill("url", " docs.example.com ");
  await submit();
  await submit();
  expect(ui.addLink.mock.calls).toEqual([[auth.id, { url: " docs.example.com ", title: null }]]);
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
  await answer(moduleLinkOf(auth, "http://docs.example.com"));
  expect(close).toHaveBeenCalledOnce();
  expect(toasts).toEqual([
    { type: "success", title: "links.toasts.created.title", message: "links.toasts.created.message" },
  ]);
  expect(dialog()).toBeNull();
  expect(document.activeElement).toBe(opener());
});

it("changes a link, its address and title as the form holds them, a title cleared as none", async () => {
  await open(docs);
  expect(title()).toBe("common.update_link");
  expect([input("url").value, input("title").value]).toEqual(["https://docs.example.com", "Docs"]);
  await fill("url", "https://docs.example.org");
  await fill("title", "");
  await submit();
  expect(ui.changeLink.mock.calls).toEqual([[docs.id, { url: "https://docs.example.org", title: null }]]);
  expect(close).not.toHaveBeenCalled();
  await answer({ ...docs, url: "https://docs.example.org", title: null, metadata: {} });
  expect(close).toHaveBeenCalledOnce();
  expect(toasts).toEqual([
    { type: "success", title: "links.toasts.updated.title", message: "links.toasts.updated.message" },
  ]);
});

it.each([
  ["empty", ""],
  ["blank", "   "],
])("refuses an address left %s before sending anything: says so under it, invalid and focused", async (_, url) => {
  await open();
  await fill("url", url);
  await act(async () => input("title").focus());
  await submit();
  expect(ui.addLink).not.toHaveBeenCalled();
  expect(input("url").getAttribute("aria-invalid")).toBe("true");
  expect(described("url")).toContain("common.errors.required");
  expect(document.activeElement).toBe(input("url"));
});

it.each([
  ["url", "invalid_format"],
  ["title", "too_long"],
] as const)("says nerve's refusal of its %s under it, which takes the focus; the dialog stays", async (field, code) => {
  await open();
  await fill("url", "docs.example.com");
  await fill("title", "Docs");
  await act(async () => opener().focus());
  await submit();
  await refuse(refusal(422, "validation_failed", [{ field, code }]));
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
  expect(input(field).getAttribute("aria-invalid")).toBe("true");
  expect(described(field)).toContain(`errors.field.${code}`);
  expect(document.activeElement).toBe(input(field));
  expect([input("url").value, input("title").value]).toEqual(["docs.example.com", "Docs"]);
});

it.each([
  ["a link the module has already", refusal(409, "planning.module_link_exists"), "errors.planning_module_link_exists"],
  ["of an archived module", refusal(409, "planning.archived"), "errors.planning_archived"],
  ["of a project whose modules are off", refusal(409, "planning.modules_disabled"), "errors.planning_modules_disabled"],
  ["it may not add", refusal(403, "forbidden"), "errors.forbidden"],
  ["while nerve is busy", refusal(503, "server_busy"), "errors.server_busy"],
])("keeps the dialog and what it holds when nerve refuses %s, saying why in a toast", async (_, error, why) => {
  await open();
  await fill("url", "docs.example.com");
  await submit();
  await refuse(error);
  expect(close).not.toHaveBeenCalled();
  expect(dialog()).not.toBeNull();
  expect(input("url").value).toBe("docs.example.com");
  expect(input("url").getAttribute("aria-invalid")).not.toBe("true");
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: why }]);
  // sent again, added: it closes then, and only then says so
  sent = heldChange<ModuleLink>();
  await submit();
  expect(ui.addLink).toHaveBeenCalledTimes(2);
  await answer(moduleLinkOf(auth, "http://docs.example.com"));
  expect(close).toHaveBeenCalledOnce();
  expect(toasts.map(({ type }) => type)).toEqual(["error", "success"]);
});

it("keeps the change of a link the store no longer holds open, saying so", async () => {
  await open(docs);
  await submit();
  await refuse(new Error("Link not found"));
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.unknown" }]);
});

it("names itself by its title and keeps the natural Tab order", async () => {
  await open();
  const positive = [...dialog()!.querySelectorAll<HTMLElement>("[tabindex]")].filter((item) => item.tabIndex > 0);
  expect(positive).toEqual([]);
  expect(title()).toBe("common.add_link");
});

it("closes on Escape and Cancel, giving the focus back; neither while what it sent is out", async () => {
  await open();
  await escape();
  expect(close).toHaveBeenCalledOnce();
  expect(document.activeElement).toBe(opener());
  await open();
  await act(async () => button("common.cancel").click());
  await settle();
  expect(close).toHaveBeenCalledTimes(2);
  close.mockReset();
  await open();
  await fill("url", "docs.example.com");
  await submit();
  await escape();
  expect(button("common.cancel").disabled).toBe(true);
  expect(close).not.toHaveBeenCalled();
  expect(dialog()).not.toBeNull();
});

it.each([
  ["is answered", (change: HeldChange<ModuleLink>) => change.answer(docs)],
  ["is refused", (change: HeldChange<ModuleLink>) => change.refuse(refusal(409, "planning.module_link_exists"))],
])("follows nothing of a link that %s once its dialog has gone with its sidebar", async (_, settleIt) => {
  await open();
  await fill("url", "docs.example.com");
  await submit();
  await act(async () => root.render(null));
  await act(async () => settleIt(sent));
  await settle();
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
});

it("follows nothing of a link answered after another tab moved this one to another account", async () => {
  await open();
  await fill("url", "docs.example.com");
  await submit();
  switchAccount();
  await answer(docs);
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
});
