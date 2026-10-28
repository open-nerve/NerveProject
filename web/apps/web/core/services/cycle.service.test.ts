/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import axios from "axios";
import { expect, it } from "vitest";
import { CycleService } from "./cycle.service";

// Transport-boundary evidence, not a real-page network claim: M6 has no entity-creation API yet.
it.each([
  ["cycleDistribution", "/api/workspaces/ws/projects/project/cycles/cycle/analytics/?type=issues"],
  ["cycleProgress", "/api/workspaces/ws/projects/project/cycles/cycle/progress/"],
] as const)("preserves_domain_loading_without_issue_lists: %s preserves its normalized URL", async (method, url) => {
  const previous = axios.defaults.adapter;
  const calls: { method: string | undefined; url: string | undefined }[] = [];
  const problem = { status: 404, detail: "M6 not connected" };
  try {
    axios.defaults.adapter = async (config) => {
      calls.push({ method: config.method, url: config.url });
      throw { response: { data: problem } };
    };
    const service = new CycleService();
    await expect(service[method]("ws", "project", "cycle")).rejects.toEqual(problem);
    expect(calls).toEqual([{ method: "get", url }]);
  } finally {
    axios.defaults.adapter = previous;
  }
});
