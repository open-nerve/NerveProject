/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { runInAction } from "mobx";
import { expect, vi } from "vitest";
import type { Asset, NewAsset, PreparedAsset } from "@nerve/api-client";
import { json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";

export const imageFile = (name = "picture.png", bytes = 4, type = "image/png") =>
  new File([new Uint8Array(bytes)], name, { type });
export const avatarTarget = { purpose: "USER_AVATAR", scope: { kind: "user" } } as const;

export function preparedAsset(id: string, input: NewAsset): PreparedAsset {
  return {
    asset: {
      ...input,
      id,
      state: "pending",
      is_bound: false,
      width: null,
      height: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
    upload: {
      method: "PUT",
      url: `/static/${id}?token=private`,
      headers: [{ name: "Content-Type", value: input.content_type }],
      expires_at: new Date(Date.now() + 60_000).toISOString(),
    },
  };
}

/** Real root, API service and PUT; only the two network edges wait for answers chosen by the test. */
export async function assetTab() {
  const tab = await issueStoreTab();
  const workspace = { ...tab.workspace, role: 20 as const };
  const project = { ...tab.project, member_role: 20 as const };
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspace]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, workspace, [project]);
  runInAction(() => {
    tab.app.instance.config = {
      product: "Nerve",
      version: "test",
      commit: "test",
      api_version: "v0",
      signup_enabled: true,
      workspace_creation_enabled: true,
      file_size_limit: 10,
    };
  });
  tab.nerve.calls.length = 0;
  const assets = new Map<string, Asset>();
  let number = 0;
  const answer = (call: Call) => {
    if (call.path === "/api/v0/assets") {
      const prepared = preparedAsset(`asset-${++number}`, call.body as NewAsset);
      assets.set(prepared.asset.id, prepared.asset);
      call.answer(json(201, prepared));
    } else if (call.path.endsWith("/complete")) {
      const id = call.path.split("/").at(-2)!;
      call.answer(json(200, { ...assets.get(id), state: "uploaded", width: 2, height: 2 }));
    }
  };
  tab.nerve.respond = answer;
  const puts: {
    file: BodyInit | null | undefined;
    signal: AbortSignal | null | undefined;
    answer: (response: Response) => void;
    fail: () => void;
  }[] = [];
  vi.stubGlobal(
    "fetch",
    (_url: RequestInfo | URL, init?: RequestInit) =>
      new Promise<Response>((resolve, reject) => {
        puts.push({
          file: init?.body,
          signal: init?.signal,
          answer: resolve,
          fail: () => reject(new TypeError("network failed")),
        });
      })
  );
  const sent = async (result: { settled: boolean }, count = 1) => {
    await until(() => puts.length >= count || result.settled, "raw PUT or rejected operation");
    expect(puts).toHaveLength(count);
    return puts[count - 1]!;
  };
  return { ...tab, workspace, project, puts, answer, sent, assets };
}
