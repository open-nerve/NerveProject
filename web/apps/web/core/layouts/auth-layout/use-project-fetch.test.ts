/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import type { ApiError } from "@/lib/api-error";
import { refusal } from "@/lib/fake-refusal";
import { fetchHanded, handed, response } from "@/lib/fake-session-swr";
import { projectOf } from "@/store/project/fake-projects";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// A page of a project fetches what its caller may read (M3 design 3.1, 7.1): the project, and its own reads only once
// nerve's read says he is a member; its wrapper shows what that read decides (3.19, 7.6). fake-session-swr.ts stands in
// for useSessionSWR and fake-store-hooks.ts for the stores. The address is acme's; beta is his other workspace.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-preferences", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-label", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-state", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));

const { useProjectFetch } = await import("./use-project-fetch");

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const member = projectOf("WEB", acme.id);
const guest = projectOf("WEB", acme.id, { member_role: 5 });
const seen = projectOf("WEB", acme.id, { member_role: null });
const archived = projectOf("WEB", acme.id, { archived_at: "2026-10-02T09:00:00Z" });
const archivedSeen = { ...archived, member_role: null };
/** The project the address names, a project of beta's, of which he is a member: the address is acme's. */
const elsewhere = projectOf("WEB", beta.id);
const notFound = refusal(404, "project.not_found");
const unreachable = refusal(503, "server_busy");

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  emptyStores();
  stores.workspaces = [acme, beta];
  stores.address = "acme";
});

describe("useProjectFetch", () => {
  it("fetches the project and, for a member, his tab bar in it, its labels, its members and its states", async () => {
    stores.projects = [member];
    response.current = { data: member };
    expect(useProjectFetch("acme", member.id)).toEqual({ kind: "member", project: member });
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["PROJECT", "p-web"],
      ["PROJECT_PREFERENCES", "p-web"],
      ["PROJECT_LABELS", "p-web"],
      ["PROJECT_MEMBERS", "p-web"],
      ["PROJECT_STATES", "p-web"],
    ]);
    await fetchHanded();
    expect(stores.fetched).toEqual([
      "the project p-web",
      "the tab bar in p-web",
      "the labels of p-web",
      "the members of p-web",
      "the states of p-web",
    ]);
  });

  it.each<{ when: string; projects: Project[]; data?: Project; error?: ApiError }>([
    { when: "he sees it and is no member", projects: [seen], data: seen },
    { when: "nerve has not read it yet, though its workspace's list has it", projects: [member] },
    { when: "nerve does not find it", projects: [], error: notFound },
    { when: "nerve cannot read it again for a member", projects: [member], data: member, error: unreachable },
    { when: "it is a project of another of his workspaces", projects: [elsewhere], data: elsewhere },
    { when: "it is archived, though he is its member", projects: [archived], data: archived },
  ])("fetches the project alone when $when", ({ projects, data, error }) => {
    stores.projects = projects;
    response.current = { data, error };
    useProjectFetch("acme", member.id);
    expect(handed.map(([fetch]) => fetch)).toEqual([["PROJECT", "p-web"], null, null, null, null]);
  });

  it.each<{ when: string; projects: Project[]; data?: Project; error?: ApiError; shows: object }>([
    {
      when: "nerve cannot read it the first time: the refusal wins over no answer yet and the store having it not",
      projects: [],
      error: unreachable,
      shows: { kind: "unavailable", retry: expect.any(Function) },
    },
    {
      when: "nerve cannot read it again: the refusal wins over its earlier answer and the store having it",
      projects: [member],
      data: member,
      error: unreachable,
      shows: { kind: "unavailable", retry: expect.any(Function) },
    },
    {
      when: "nerve no longer finds it: the refusal wins over its earlier answer and the store having it",
      projects: [member],
      data: member,
      error: notFound,
      shows: { kind: "not-found" },
    },
    {
      when: "nerve does not find it: the refusal wins over no answer yet and the store having it",
      projects: [member],
      error: notFound,
      shows: { kind: "not-found" },
    },
    {
      when: "nerve has not answered yet: no answer wins over the store having it",
      projects: [member],
      shows: { kind: "loading" },
    },
    {
      when: "nerve has not answered yet: no answer wins over the store having it not",
      projects: [],
      shows: { kind: "loading" },
    },
    {
      when: "he has just joined: the store's role, nerve's answer to the join, wins over SWR's read from before it",
      projects: [member],
      data: seen,
      shows: { kind: "member", project: member },
    },
    {
      when: "the store no longer gives it: the store wins over nerve's answer having it",
      projects: [],
      data: member,
      shows: { kind: "not-found" },
    },
    {
      when: "it is a project of another of his workspaces: the address's workspace wins over his role in it",
      projects: [elsewhere],
      data: elsewhere,
      shows: { kind: "not-found" },
    },
    {
      when: "it is archived: archived wins over his membership",
      projects: [archived],
      data: archived,
      shows: { kind: "archived", project: archived },
    },
    {
      when: "it is archived and he sees it, no member: archived wins over the join",
      projects: [archivedSeen],
      data: archivedSeen,
      shows: { kind: "archived", project: archivedSeen },
    },
    {
      when: "nerve no longer finds it: the refusal wins over the store having it archived",
      projects: [archived],
      data: archived,
      error: notFound,
      shows: { kind: "not-found" },
    },
    {
      when: "nerve cannot read it again: the refusal wins over the store having it archived",
      projects: [archived],
      data: archived,
      error: unreachable,
      shows: { kind: "unavailable", retry: expect.any(Function) },
    },
    {
      when: "nerve has not answered yet: no answer wins over the store having it archived",
      projects: [archived],
      shows: { kind: "loading" },
    },
    { when: "he sees it, no member", projects: [seen], data: seen, shows: { kind: "not-member", project: seen } },
    { when: "he is its guest", projects: [guest], data: guest, shows: { kind: "member", project: guest } },
    { when: "he is its member", projects: [member], data: member, shows: { kind: "member", project: member } },
  ])("shows the wrapper what to render when $when", ({ projects, data, error, shows }) => {
    stores.projects = projects;
    response.current = { data, error };
    expect(useProjectFetch("acme", member.id)).toEqual(shows);
  });

  it("decides by the address, not by the stores' current workspace, a render behind it: another's project is not found", () => {
    // the stores' address still names beta, of which the project is; the page's address is acme's
    stores.address = "beta";
    stores.projects = [elsewhere];
    response.current = { data: elsewhere };
    expect(useProjectFetch("acme", elsewhere.id)).toEqual({ kind: "not-found" });
    expect(handed.map(([fetch]) => fetch)).toEqual([["PROJECT", "p-web"], null, null, null, null]);
  });

  it("reads the project again when the page's retry asks", () => {
    const mutate = vi.fn(() => Promise.resolve(undefined));
    response.current = { error: unreachable, mutate };
    const access = useProjectFetch("acme", member.id);
    if (access.kind === "unavailable") access.retry();
    expect(mutate).toHaveBeenCalledOnce();
  });
});
