// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { bodyImagesTab } from "./fake-body-images";
import { contentAssetOf, imageFile, watchLegacyFileTraffic } from "@/store/asset/fake-assets";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { loadProjects } from "@/store/project/fake-projects";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "images" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "images" }) }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/editor/rich-text", () => import("@/components/editor/rich-text/editor-implementation"));
vi.mock("@/components/editor/lite-text", () => import("@/components/editor/lite-text/editor-implementation"));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ issueDetailSidebarCollapsed: true }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-window-size", () => ({ default: () => [1024, 768] }));
vi.mock("@/hooks/use-reload-confirmation", () => ({ default: () => ({ setShowAlert: () => undefined }) }));
vi.mock("@/components/issues/issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("@/components/issues/issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("@/components/issues/issue-detail/sidebar", () => ({ IssueDetailsSidebar: () => null }));
vi.mock("@/components/issues/peek-overview/properties", () => ({ PeekOverviewProperties: () => null }));
vi.mock("@/components/issues/peek-overview/header", () => ({ IssuePeekOverviewHeader: () => null }));
vi.mock("@/components/issues/issue-detail-widgets/action-buttons", () => ({
  IssueDetailWidgetActionButtons: () => null,
}));
vi.mock("@/components/issues/issue-detail-widgets/issue-detail-widget-modals", () => ({
  IssueDetailWidgetModals: () => null,
}));
let tab: Awaited<ReturnType<typeof bodyImagesTab>>;
let legacy: ReturnType<typeof watchLegacyFileTraffic>;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.stubGlobal(
    "ClipboardEvent",
    class extends Event {
      readonly clipboardData = null;
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  const unlaidOut = document.createElement("div");
  // oxlint-disable-next-line no-extend-native -- declares what the test environment lacks; tests only
  Object.defineProperties(Range.prototype, {
    getClientRects: { configurable: true, value: () => unlaidOut.getClientRects() },
    getBoundingClientRect: { configurable: true, value: () => unlaidOut.getBoundingClientRect() },
  });
  tab = await bodyImagesTab();
  legacy = watchLegacyFileTraffic(tab);
});
afterEach(async () => {
  const leaks = legacy.leaks();
  await tab?.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
  expect(leaks).toEqual([]);
});
async function uploadImage() {
  await tab.paste(undefined, [imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 1, "image PUT");
  expect(tab.assets.values().next().value?.purpose).toBe("ISSUE_DESCRIPTION");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(0);
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  await waitForPickerDOM(() => !!document.querySelector("img.image-component"), "confirmed image");
  return [...tab.assets.keys()][0]!;
}
it.each(["detail", "peek"] as const)("%s saves only confirmed UUIDs and does not save grant renewal", async (kind) => {
  await tab.render(kind);
  const id = await uploadImage();
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  expect(tab.patches()[0]!.body).toMatchObject({ description_html: expect.stringContaining(`src="${id}"`) });
  expect(tab.patches()[0]!.body).not.toHaveProperty("skip_activity");
  expect(JSON.stringify(tab.patches()[0]!.body)).not.toContain("grant=");
  const first = document.querySelector("img.image-component")!.getAttribute("src");
  await act(async () => vi.advanceTimersByTimeAsync(60_100));
  await waitForPickerDOM(
    () => document.querySelector("img.image-component")?.getAttribute("src") !== first,
    "fresh grant"
  );
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
});
it.each(["detail", "peek"] as const)(
  "%s retains a rejected body across attachment verification and requires explicit recovery",
  async (kind) => {
    await tab.render(kind);
    const id = await uploadImage();
    const accept = tab.control.patch;
    tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
    await tab.debounce();
    expect(tab.patches()).toHaveLength(1);
    let pending: Call | undefined;
    tab.control.core = (call) => {
      pending = call;
    };
    await act(async () => {
      [...document.querySelectorAll<HTMLButtonElement>("button")]
        .find((button) => button.textContent?.includes("content_files.title"))!
        .click();
    });
    await tab.select([new File(["note"], "note.txt", { type: "text/plain" })]);
    await tab.click("content_files.upload");
    await waitForPickerDOM(() => tab.puts.length === 2, "attachment PUT");
    await act(async () => tab.puts[1]!.answer(new Response(null, { status: 200 })));
    await waitForPickerDOM(() => !!pending, "owner pending");
    expect(tab.editor()).toBeNull();
    await act(async () => pending!.answer(problem(503, "server_busy")));
    tab.control.core = (call) => call.answer(json(200, { ...tab.canonical(), attachment_count: 1 }));
    await tab.click("common.retry");
    await waitForPickerDOM(() => !!tab.editor(), "fresh body");
    expect(tab.editor()!.innerHTML).toContain(id);
    await tab.debounce();
    expect(tab.patches()).toHaveLength(1);
    tab.control.patch = accept;
    await tab.click("content_files.input.save");
    await waitForPickerDOM(() => tab.patches().length === 2, "explicit save");
    expect(tab.patches()[1]!.body).toMatchObject({ description_html: expect.stringContaining(id) });
  }
);

it("a concurrent canonical body requires an explicit choice and does not silently replace retained HTML", async () => {
  await tab.render();
  const id = await uploadImage();
  tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
  await tab.debounce();
  tab.control.core = (call) =>
    call.answer(json(200, { ...tab.canonical(), description_html: '<p data-id="other">Another tab</p>' }));
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  });
  await waitForPickerDOM(() => !!tab.editor(), "fresh owner");
  expect(tab.editor()!.innerHTML).toContain(id);
  expect(document.body.textContent).toContain("content_files.input.conflict");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  await tab.click("content_files.input.latest");
  expect(tab.editor()!.textContent).toBe("Another tab");
  expect(document.querySelector("img.image-component")).toBeNull();
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
});

it("permission ABA removes retained image input and aborts its unresolved upload permanently", async () => {
  await tab.render();
  await tab.paste(undefined, [imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 1, "pending image");
  const respond = tab.nerve.respond;
  tab.nerve.respond = undefined;
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  });
  tab.nerve.respond = respond;
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
    tab.puts[0]!.answer(new Response(null, { status: 200 }));
  });
  await waitForPickerDOM(() => !!tab.editor(), "new owner");
  expect(tab.editor()!.querySelector("image-component,.image-upload-component")).toBeNull();
  await tab.debounce();
  expect(tab.patches()).toHaveLength(0);
});

it.each(["new-comment", "edit-comment"] as const)(
  "%s writes confirmed description UUIDs without adding them to attachment_ids",
  async (kind) => {
    await tab.render(kind);
    await tab.paste(undefined, [imageFile()]);
    await waitForPickerDOM(() => tab.puts.length === 1, "comment image PUT");
    const id = [...tab.assets.keys()][0]!;
    expect(tab.assets.get(id)).toMatchObject({
      purpose: "COMMENT_DESCRIPTION",
      scope: {
        kind: kind === "new-comment" ? "new_comment" : "comment",
        id: kind === "new-comment" ? tab.issue.id : tab.comment.id,
      },
    });
    const writes = () =>
      tab.nerve.calls.filter(
        (call) => /\/comments|\/issue-comments\//.test(call.path) && ["POST", "PATCH"].includes(call.method)
      );
    const button = kind === "new-comment" ? "common.comment" : "save";
    expect(tab.button(button).disabled).toBe(true);
    await tab.click(button);
    expect(writes()).toHaveLength(0);
    await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
    await waitForPickerDOM(() => !!document.querySelector("img.image-component"), "confirmed comment image");
    await waitForPickerDOM(() => !tab.button(button).disabled, "ready comment action");
    await tab.click(button);
    await waitForPickerDOM(() => writes().length === 1, "comment save");
    expect(writes()[0]!.body).toMatchObject({ comment_html: expect.stringContaining(id) });
    if (kind === "new-comment") expect(writes()[0]!.body).toMatchObject({ attachment_ids: [] });
    else expect(writes()[0]!.body).not.toHaveProperty("attachment_ids");
    expect(JSON.stringify(writes()[0]!.body)).not.toContain("grant=");
    expect(tab.nerve.calls.some((call) => /bulk|file-assets|assets\/v2/.test(call.path))).toBe(false);
  }
);

it("a preserved comment must reauthorize its image before an explicit resend", async () => {
  await tab.render("new-comment");
  await tab.paste(undefined, [imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 1, "comment image");
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
  await waitForPickerDOM(() => !tab.button("common.comment").disabled, "ready comment");
  const id = [...tab.assets.keys()][0]!;
  tab.control.comment = (call) => call.answer(problem(422, "validation_failed"));
  await tab.click("common.comment");
  const posts = () => tab.nerve.calls.filter((call) => call.path.endsWith("/comments") && call.method === "POST");
  expect(posts()).toHaveLength(1);
  tab.control.asset = (call) => call.answer(problem(404, "asset.not_found"));
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  });
  await waitForPickerDOM(() => !!tab.editor(), "retained comment");
  expect(tab.editor()!.innerHTML).not.toContain("grant=");
  await waitForPickerDOM(
    () => document.body.textContent?.includes("errors.asset_not_found") ?? false,
    "reauthorized image refusal"
  );
  await waitForPickerDOM(() => !tab.button("common.comment").disabled, "explicit resend action");
  await tab.click("common.comment");
  await tab.debounce();
  expect(posts()).toHaveLength(1);
  expect(tab.nerve.to(`/api/v0/assets/${id}`).length).toBeGreaterThan(1);
});

it("history can resolve an image removed from the current body and exposes no write capability", async () => {
  const id = "22222222-2222-4222-8222-222222222222";
  tab.assets.set(id, { ...contentAssetOf(id, tab.hosts.issue, "description"), is_bound: true });
  tab.control.historyHTML = `<p data-id="old">Past body</p><image-component id="old-image" src="${id}"></image-component>`;
  expect(tab.canonical().description_html).not.toContain(id);
  await tab.render("history");
  await waitForPickerDOM(() => !!document.querySelector("img.image-component"), "historical image");
  expect(tab.control.historyImages).toBeDefined();
  expect(tab.control.historyImages?.write).toBeUndefined();
  expect(document.querySelector('input[type="file"]')).toBeNull();
  expect(document.querySelector<HTMLImageElement>("img.image-component")!.src).toContain(id);
  await tab.debounce();
  expect(tab.patches()).toHaveLength(0);
  expect(tab.nerve.to(`/api/v0/assets/${id}`)).toHaveLength(1);
});

it("an unfinished last image is retained but never flushed when its editor unmounts", async () => {
  await tab.render();
  await tab.paste(undefined, [imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 1, "unfinished image");
  await act(async () => tab.renderer.render(null));
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  await tab.debounce();
  expect(tab.patches()).toHaveLength(0);
});

it("a text-only preserved description resumes autosave when editing continues", async () => {
  await tab.render();
  const accept = tab.control.patch;
  tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
  await tab.paste("<p>Local text</p>");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  });
  await waitForPickerDOM(() => !!tab.editor(), "preserved text");
  expect(document.body.textContent).toContain("content_files.input.preserved");
  tab.control.patch = accept;
  await tab.paste("<p>More text</p>");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(2);
  expect(tab.patches()[1]!.body).toMatchObject({ description_html: expect.stringContaining("More text") });
});

it("editing preserved text cannot bypass the explicit image verification before resaving", async () => {
  await tab.render();
  await uploadImage();
  tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
  await tab.debounce();
  tab.control.asset = (call) => call.answer(problem(404, "asset.not_found"));
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  });
  await waitForPickerDOM(() => !!tab.editor(), "preserved description");
  await tab.paste("<p>Revised retained text</p>");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  expect(document.body.textContent).toContain("content_files.input.preserved");
  await tab.click("content_files.input.save");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  expect(document.body.textContent).toContain("content_files.image.unavailable");
});

it("saving preserved input while an image uploads keeps the editor and the upload, and saves once it finishes", async () => {
  await tab.render();
  const first = await uploadImage();
  const accept = tab.control.patch;
  tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
  await tab.debounce();
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  });
  await waitForPickerDOM(() => !!tab.editor(), "preserved description");
  await tab.paste(undefined, [imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 2, "second image PUT");
  const field = tab.editor()!;
  await tab.click("content_files.input.save");
  expect(tab.editor()).toBe(field);
  expect(tab.puts[1]!.signal?.aborted).toBe(false);
  expect(document.body.textContent).toContain("content_files.image.unfinished");
  expect(document.body.textContent).not.toContain("content_files.image.unavailable");
  await act(async () => tab.puts[1]!.answer(new Response(null, { status: 200 })));
  await waitForPickerDOM(() => document.querySelectorAll("img.image-component").length === 2, "second image");
  const second = [...tab.assets.keys()].find((id) => id !== first)!;
  await tab.click("content_files.input.save");
  await waitForPickerDOM(() => tab.patches().length === 2, "refused save");
  expect(tab.editor()).toBe(field);
  expect(field.innerHTML).toContain(second);
  tab.control.patch = accept;
  await tab.click("common.retry");
  await waitForPickerDOM(() => tab.patches().length === 3, "retried save");
  expect(tab.patches()[2]!.body).toMatchObject({ description_html: expect.stringContaining(second) });
  expect(tab.patches()[2]!.body).toMatchObject({ description_html: expect.stringContaining(first) });
});

it("verifying preserved input blocks the same editor while it waits, and a refused image keeps the input", async () => {
  await tab.render();
  const id = await uploadImage();
  tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
  await tab.debounce();
  const reads: Call[] = [];
  tab.control.asset = (call) => {
    reads.push(call);
  };
  await act(async () => {
    await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
  });
  await waitForPickerDOM(() => !!tab.editor(), "preserved description");
  const field = tab.editor()!;
  await tab.click("content_files.input.save");
  expect(reads.map((call) => call.path)).toContain(`/api/v0/assets/${id}`);
  expect(tab.editor()).toBe(field);
  expect(field.closest("[inert]")?.getAttribute("aria-busy")).toBe("true");
  await act(async () => {
    for (const call of reads.splice(0)) call.answer(problem(404, "asset.not_found"));
  });
  await waitForPickerDOM(() => document.body.textContent!.includes("content_files.image.unavailable"), "refused image");
  expect(tab.editor()).toBe(field);
  expect(field.closest("[inert]")).toBeNull();
  expect(field.querySelector(".node-imageComponent [role=alert]")?.textContent).toContain("errors.asset_not_found");
  expect(tab.patches()).toHaveLength(1);
});

it.each(["detail", "peek"] as const)(
  "%s keeps a refused image body and an unsaved title through a failed read and a concurrent body, then permission ABA clears both",
  async (kind) => {
    await tab.render(kind);
    const id = await uploadImage();
    tab.control.patch = (call) => call.answer(problem(422, "validation_failed"));
    await tab.debounce();
    expect(tab.patches()).toHaveLength(1);
    const storedTitle = tab.canonical().name;
    let pending: Call | undefined;
    tab.control.core = (call) => {
      pending = call;
    };
    await act(async () => {
      [...document.querySelectorAll<HTMLButtonElement>("button")]
        .find((button) => button.textContent?.includes("content_files.title"))!
        .click();
    });
    await tab.select([new File(["note"], "note.txt", { type: "text/plain" })]);
    await tab.click("content_files.upload");
    await waitForPickerDOM(() => tab.puts.length === 2, "attachment PUT");
    // Typed after the attachment PUT starts, so its debounce cannot run before the owner disappears.
    await tab.typeTitle("Unsaved local title");
    await act(async () => tab.puts[1]!.answer(new Response(null, { status: 200 })));
    await waitForPickerDOM(() => !!pending, "owner pending");
    expect(tab.editor()).toBeNull();
    expect(tab.titleField()).toBeNull();
    await act(async () => pending!.answer(problem(503, "server_busy")));
    tab.control.core = (call) =>
      call.answer(
        json(200, { ...tab.canonical(), attachment_count: 1, description_html: '<p data-id="other">Another tab</p>' })
      );
    await tab.click("common.retry");
    await waitForPickerDOM(() => !!tab.editor(), "fresh owner");
    expect(tab.titleField()!.value).toBe("Unsaved local title");
    expect(tab.editor()!.innerHTML).toContain(id);
    expect(document.body.textContent).toContain("content_files.input.conflict");
    expect(document.body.textContent).toContain("content_files.input.preserved");
    await tab.debounce();
    expect(tab.patches()).toHaveLength(1);
    expect(tab.nerve.calls.filter((call) => call.path === "/api/v0/assets" && call.method === "POST")).toHaveLength(2);
    const respond = tab.nerve.respond;
    tab.nerve.respond = undefined;
    await act(async () => {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    });
    tab.nerve.respond = respond;
    if (kind === "peek") {
      // Losing the project closes the peek; reopening it is a new owner, never the retained one.
      expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
      await act(async () =>
        tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: tab.issue.id })
      );
    }
    await act(async () => {
      await tab.app.issue.issues.refreshAuthoritative(tab.issue.id);
    });
    await waitForPickerDOM(() => !!tab.editor(), "owner after ABA");
    expect(tab.titleField()!.value).toBe(storedTitle);
    expect(tab.editor()!.textContent).toBe("Another tab");
    expect(tab.editor()!.querySelector("image-component,.image-upload-component")).toBeNull();
    expect(document.body.textContent).not.toContain("content_files.input.preserved");
    expect(document.body.textContent).not.toContain("content_files.input.conflict");
    await tab.debounce();
    expect(tab.patches()).toHaveLength(1);
  }
);

it("replacing a saved image saves the new identity as an ordinary versioned edit without deleting the old asset", async () => {
  await tab.render();
  const first = await uploadImage();
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  await tab.removeImage(first);
  await tab.paste(undefined, [imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 2, "replacement PUT");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  await act(async () => tab.puts[1]!.answer(new Response(null, { status: 200 })));
  const second = [...tab.assets.keys()][1]!;
  await waitForPickerDOM(
    () => document.querySelector("img.image-component")?.getAttribute("src")?.includes(second) ?? false,
    "replacement image"
  );
  await tab.debounce();
  expect(tab.patches()).toHaveLength(2);
  const body = tab.patches()[1]!.body as { description_html: string };
  expect(body.description_html).toContain(`src="${second}"`);
  expect(body.description_html).not.toContain(first);
  expect(body).not.toHaveProperty("skip_activity");
  expect(tab.nerve.calls.some((call) => call.method === "DELETE")).toBe(false);
});

it("reordering two saved images saves the new order as an ordinary versioned edit without copying either", async () => {
  await tab.render();
  await tab.paste(undefined, [imageFile(), imageFile()]);
  await waitForPickerDOM(() => tab.puts.length === 2, "two image PUTs");
  await act(async () => {
    for (const put of tab.puts) put.answer(new Response(null, { status: 200 }));
  });
  await waitForPickerDOM(() => document.querySelectorAll("img.image-component").length === 2, "two confirmed images");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  const saved = (tab.patches()[0]!.body as { description_html: string }).description_html;
  const [upper, lower] = [...tab.assets.keys()].toSorted((left, right) => saved.indexOf(left) - saved.indexOf(right));
  expect(saved.indexOf(upper!)).toBeGreaterThan(-1);
  await tab.moveImageBefore(lower!, upper!);
  await tab.debounce();
  expect(tab.patches()).toHaveLength(2);
  const reordered = tab.patches()[1]!.body as { description_html: string };
  expect(reordered.description_html.indexOf(lower!)).toBeLessThan(reordered.description_html.indexOf(upper!));
  expect(reordered).not.toHaveProperty("skip_activity");
  expect(tab.nerve.calls.some((call) => call.path.endsWith("/duplicate"))).toBe(false);
});

it("a stored image without its block identity gains one in a single migration write", async () => {
  const id = "33333333-3333-4333-8333-333333333333";
  tab.assets.set(id, { ...contentAssetOf(id, tab.hosts.issue, "description"), is_bound: true });
  await tab.setBody(`<p data-id="base">Original body</p><image-component src="${id}"></image-component>`);
  await tab.render();
  await waitForPickerDOM(() => !!document.querySelector("img.image-component"), "stored image");
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
  expect(tab.patches()[0]!.body).toEqual({
    description_html: expect.stringContaining(`src="${id}"`),
    skip_activity: true,
  });
  const saved = (tab.patches()[0]!.body as { description_html: string }).description_html;
  const image = new DOMParser().parseFromString(saved, "text/html").querySelector("image-component");
  expect(image?.getAttribute("id") ?? image?.getAttribute("data-id")).toMatch(/^[0-9a-f-]{36}$/);
  await tab.debounce();
  expect(tab.patches()).toHaveLength(1);
});
