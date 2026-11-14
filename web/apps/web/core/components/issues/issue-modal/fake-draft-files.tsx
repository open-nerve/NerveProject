/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { expect, vi, type Mock } from "vitest";
import type {
  Asset,
  Attachment,
  Draft,
  DraftCreate,
  DraftUpdate,
  DuplicateAsset,
  Issue,
  IssueDetail,
  NewAsset,
} from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import { contentAssetOf, contentAssetTab } from "@/store/asset/fake-assets";
import { pickerDOM, waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { CreateUpdateIssueModalBase } from "./base";
import type { IssuesModalProps } from "./modal";

const now = () => new Date().toISOString();
/** The draft modal's files section, by its accessible name. */
const files = () => document.querySelector<HTMLElement>('section[aria-label="content_files.title"]');
const imageID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
/** Real root, modal, draft and asset services; tests choose only what the network edges answer. */
export async function draftFilesTab() {
  const tab = await pickerDOM(await contentAssetTab());
  const slug = tab.workspace.slug;
  let sequence = 0;
  let grants = 0;
  const id = () => `22222222-2222-4222-8222-${String(++sequence).padStart(12, "0")}`;
  let drafts: Draft[] = [];
  const attachments = new Map<string, Attachment[]>();
  const issues: IssueDetail[] = [];
  const bindImages = (draft: Draft) => {
    for (const [key, asset] of tab.assets)
      if (
        asset.scope.kind === "draft" &&
        asset.scope.id === draft.id &&
        draft.payload.description_html?.includes(`src="${key}"`)
      )
        tab.assets.set(key, { ...asset, is_bound: true });
  };
  const control = {
    /** Calls a test answers itself stay pending in `held`. */
    hold: (_call: Call) => false,
    create: (call: Call) => {
      const body = call.body as DraftCreate;
      const row: Draft = {
        id: id(),
        workspace_id: tab.workspace.id,
        project_id: body.project_id ?? null,
        payload: body.payload,
        created_at: now(),
        updated_at: now(),
      };
      drafts = [row, ...drafts];
      call.answer(json(201, row));
    },
    update: (call: Call, draft: Draft) => {
      const body = call.body as DraftUpdate;
      const row: Draft = {
        ...draft,
        ...(Object.hasOwn(body, "project_id") ? { project_id: body.project_id ?? null } : {}),
        payload: body.payload ?? draft.payload,
        updated_at: now(),
      };
      drafts = drafts.map((stored) => (stored.id === row.id ? row : stored));
      bindImages(row);
      call.answer(json(200, row));
    },
    publish: (call: Call, draft: Draft) => {
      if (!draft.project_id) return call.answer(problem(422, "draft.project_required"));
      if ([...tab.assets.values()].some((asset) => asset.scope.id === draft.id && asset.state !== "uploaded"))
        return call.answer(problem(409, "asset.not_uploaded"));
      drafts = drafts.filter((stored) => stored.id !== draft.id);
      const issue = {
        ...issueDetailOf(id(), draft.project_id),
        workspace_id: draft.workspace_id,
        name: draft.payload.name ?? "",
        description_html: draft.payload.description_html ?? "<p></p>",
        // nerve publishes the work item in the draft's cycle and modules
        cycle_id: draft.payload.cycle_id ?? null,
        module_ids: draft.payload.module_ids ?? [],
      };
      issues.push(issue);
      call.answer(json(200, issue));
    },
    duplicate: (call: Call, source: Asset) => {
      const body = call.body as DuplicateAsset;
      const row: Asset = { ...source, ...body, id: id(), is_bound: false, created_at: now(), updated_at: now() };
      tab.assets.set(row.id, row);
      call.answer(json(201, row));
    },
    prepare: (call: Call) => {
      const row: Asset = {
        ...(call.body as NewAsset),
        id: id(),
        state: "pending",
        is_bound: false,
        width: null,
        height: null,
        created_at: now(),
        updated_at: now(),
      };
      tab.assets.set(row.id, row);
      call.answer(
        json(201, {
          asset: row,
          upload: {
            method: "PUT",
            url: `/static/${row.id}?upload=1`,
            headers: [{ name: "Content-Type", value: row.content_type }],
            expires_at: new Date(Date.now() + 60_000).toISOString(),
          },
        })
      );
    },
  };
  const held: Call[] = [];
  tab.nerve.respond = (call) => {
    if (control.hold(call)) return void held.push(call);
    const [, , , resource, key, rest] = call.path.split("/");
    if (call.path === `/api/v0/workspaces/${slug}/drafts`) {
      if (call.method === "POST") return control.create(call);
      return call.answer(json(200, { data: drafts, total: drafts.length, next_cursor: null }));
    }
    if (resource === "drafts") {
      const draft = drafts.find((row) => row.id === key);
      if (!draft) return call.answer(problem(404, "draft.not_found"));
      if (call.method === "PATCH") return control.update(call, draft);
      if (rest === "publish") return control.publish(call, draft);
      if (rest === "attachments")
        return call.answer(json(200, { data: attachments.get(draft.id) ?? [], next_cursor: null }));
      const unfinished = [...tab.assets.values()].filter(
        (asset) => asset.scope.kind === "draft" && asset.scope.id === draft.id && !asset.is_bound
      );
      return call.answer(json(200, { data: unfinished, next_cursor: null }));
    }
    if (call.path === "/api/v0/assets") return control.prepare(call);
    if (resource === "assets") {
      const asset = tab.assets.get(key!);
      if (!asset) return call.answer(problem(404, "asset.not_found"));
      if (rest === "duplicate") return control.duplicate(call, asset);
      if (rest === "content")
        return call.answer(
          json(200, {
            url: `/static/${key}?grant=${++grants}`,
            expires_at: new Date(Date.now() + 60_000).toISOString(),
          })
        );
      if (rest === "complete") {
        const row: Asset = {
          ...asset,
          state: "uploaded",
          is_bound: asset.purpose.endsWith("ATTACHMENT"),
          width: 2,
          height: 2,
        };
        tab.assets.set(row.id, row);
        if (row.purpose === "DRAFT_ISSUE_ATTACHMENT")
          attachments.set(row.scope.id!, [
            ...(attachments.get(row.scope.id!) ?? []),
            {
              id: row.id,
              name: row.name,
              content_type: row.content_type,
              size: row.size,
              created_at: row.created_at,
              created_by_id: "u-actor",
            },
          ]);
        return call.answer(json(200, row));
      }
      if (call.method === "DELETE") {
        tab.assets.delete(asset.id);
        return call.answer(new Response(null, { status: 204 }));
      }
      return call.answer(json(200, asset));
    }
    if (call.method === "POST" && resource === "projects" && rest === "issues") {
      const issue = {
        ...issueDetailOf(id(), key!),
        ...(call.body as Partial<IssueDetail>),
        workspace_id: tab.workspace.id,
      };
      issues.push(issue);
      return call.answer(json(201, issue));
    }
    if (call.method === "GET") call.answer(json(200, { data: [], next_cursor: null }));
  };
  /** A stored draft whose description already references bound images of its own. */
  const seed = async (row: Partial<Draft> = {}, images: string[] = []) => {
    const draft: Draft = {
      ...tab.draft,
      payload: { name: "Draft", description_html: '<p data-id="base">Draft body</p>' },
      ...row,
    };
    for (const image of images)
      tab.assets.set(image, {
        ...contentAssetOf(
          image,
          { kind: "draft", workspaceID: tab.workspace.id, projectID: null, draftID: draft.id },
          "description"
        ),
      });
    drafts = [draft, ...drafts.filter((stored) => stored.id !== draft.id)];
    await tab.readDrafts(drafts);
    tab.nerve.calls.length = 0;
    return draft;
  };
  const render = (props: IssuesModalProps) =>
    act(async () =>
      tab.renderer.render(
        <StoreContext.Provider value={tab.app}>
          <CreateUpdateIssueModalBase
            fetchIssueDetails={false}
            storeType={EIssuesStoreType.WORKSPACE_DRAFT}
            withDraftIssueWrapper={false}
            {...props}
          />
        </StoreContext.Provider>
      )
    );
  const close: Mock<() => void> = vi.fn();
  const confirmed: Mock<(row: Draft | Issue) => Promise<void>> = vi.fn(async () => {});
  /** Opens the stored draft as the draft list does, after its canonical read. */
  const openDraft = async (draftID: string, moveToIssue = false) => {
    const draft = tab.app.issue.workspaceDraftIssues.getIssueById(draftID, slug);
    expect(draft, "canonical draft").toBeDefined();
    await render({ isOpen: true, isDraft: true, draft, moveToIssue, onClose: close, onSubmit: confirmed });
  };
  const calls = (method: string, path: string | RegExp) =>
    tab.nerve.calls.filter(
      (call) => call.method === method && (typeof path === "string" ? call.path === path : path.test(call.path))
    );
  const writes = () => tab.nerve.calls.filter((call) => call.method !== "GET");
  const waitFor = (done: () => boolean, what: string) => waitForPickerDOM(done, what);
  const submit = () =>
    act(async () =>
      document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
  const chooseProject = (value: string | null) =>
    act(async () => {
      const select = document.querySelector<HTMLSelectElement>('select[aria-label="project"]')!;
      expect(select).not.toBeNull();
      Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value")!.set!.call(select, value ?? "");
      select.dispatchEvent(new Event("change", { bubbles: true }));
    });
  const uploadAttachment = async (file = new File(["text"], "notes.txt", { type: "text/plain" })) => {
    await tab.select([file], files()!);
    await tab.click("content_files.upload", files()!);
  };
  return {
    ...tab,
    control,
    held,
    attachments,
    issues,
    drafts: () => drafts,
    seed,
    render,
    close,
    confirmed,
    openDraft,
    calls,
    writes,
    files,
    waitFor,
    submit,
    chooseProject,
    uploadAttachment,
    imageID,
  };
}
