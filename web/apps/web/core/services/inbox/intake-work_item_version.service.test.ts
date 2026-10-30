/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import axios from "axios";
import { expect, it } from "vitest";
import { IntakeWorkItemVersionService } from "./intake-work_item_version.service";
import type {
  TIntakeDescriptionVersionDetails,
  TIntakeDescriptionVersionsListResponse,
} from "./intake-description-version";
const row: TIntakeDescriptionVersionDetails = {
  id: "v",
  created_at: "created",
  created_by: null,
  last_saved_at: "saved",
  owned_by: "owner",
  project: "A",
  updated_at: "updated",
  updated_by: null,
  description_html: null,
  description_json: null,
  description_stripped: null,
};
const page: TIntakeDescriptionVersionsListResponse = {
  cursor: "cursor",
  next_cursor: null,
  next_page_results: false,
  page_count: 1,
  prev_cursor: null,
  prev_page_results: false,
  results: [row],
  total_pages: 1,
  total_results: 1,
};
it.each([false, true])("preserves exact M7 paths, absent body and nullable DTOs (refused=%s)", async (refused) => {
  const previous = axios.defaults.adapter;
  const calls: unknown[] = [];
  const reason = { code: "not_connected", status: 404 };
  try {
    axios.defaults.adapter = async (config) => {
      calls.push([config.method, config.url, config.data]);
      if (refused) throw { response: { data: reason } };
      return { data: config.url?.endsWith("/v/") ? row : page, status: 200, statusText: "OK", headers: {}, config };
    };
    const service = new IntakeWorkItemVersionService();
    const list = service.listDescriptionVersions("ws", "A", "intake");
    if (refused) await expect(list).rejects.toBe(reason);
    else expect(await list).toBe(page);
    const get = service.retrieveDescriptionVersion("ws", "A", "intake", "v");
    if (refused) await expect(get).rejects.toBe(reason);
    else expect(await get).toBe(row);
    expect(calls).toEqual([
      ["get", "/api/workspaces/ws/projects/A/intake-work-items/intake/description-versions/", undefined],
      ["get", "/api/workspaces/ws/projects/A/intake-work-items/intake/description-versions/v/", undefined],
    ]);
  } finally {
    axios.defaults.adapter = previous;
  }
});
