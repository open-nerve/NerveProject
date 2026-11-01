/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import axios from "axios";
import { expect, it } from "vitest";
import { WorkspaceService } from "./workspace.service";

// A transport contract only: the M7 endpoint is not implemented, so no page/remote success is claimed here.
it("retainsViewTransportWhileSearchIsRetired", async () => {
  const previous = axios.defaults.adapter;
  const calls: { method: string | undefined; url: string | undefined; data: unknown }[] = [];
  const reply = { transport: "unchanged view response" };
  try {
    axios.defaults.adapter = async (config) => {
      calls.push({ method: config.method, url: config.url, data: config.data });
      return { config, data: reply, status: 200, statusText: "OK", headers: {} };
    };
    const service = new WorkspaceService();
    expect(await service.createView("ws", { name: "Keep M7" })).toBe(reply);
    expect(await service.updateView("ws", "view", { name: "Renamed" })).toBe(reply);
    expect(await service.deleteView("ws", "view")).toBe(reply);
    expect(await service.getAllViews("ws")).toBe(reply);
    expect(calls).toEqual([
      { method: "post", url: "/api/workspaces/ws/views/", data: JSON.stringify({ name: "Keep M7" }) },
      { method: "patch", url: "/api/workspaces/ws/views/view/", data: JSON.stringify({ name: "Renamed" }) },
      { method: "delete", url: "/api/workspaces/ws/views/view/", data: undefined },
      { method: "get", url: "/api/workspaces/ws/views/", data: undefined },
    ]);
    expect("searchWorkspace" in service).toBe(false);
  } finally {
    axios.defaults.adapter = previous;
  }
});
it.each(["createView", "updateView", "deleteView", "getAllViews"] as const)(
  "%s preserves the existing M7 problem body without inventing a result",
  async (method) => {
    const previous = axios.defaults.adapter;
    const problem = { status: 404, code: "not_found" };
    try {
      axios.defaults.adapter = async () => {
        throw { response: { data: problem } };
      };
      const service = new WorkspaceService();
      const calls = {
        createView: () => service.createView("ws", { name: "View" }),
        updateView: () => service.updateView("ws", "view", { name: "View" }),
        deleteView: () => service.deleteView("ws", "view"),
        getAllViews: () => service.getAllViews("ws"),
      };
      await expect(calls[method]()).rejects.toBe(problem);
    } finally {
      axios.defaults.adapter = previous;
    }
  }
);
