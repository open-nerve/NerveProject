// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { MemberUser } from "@nerve/api-client";
import { useActorName } from "./use-actor-name";

// Who made a change, as a page shows him (P4 spec R22, the rule IssueUser keeps): no account is nerve itself (an
// automatic archive), shown by the site's name; an account the workspace's members no longer list (they list the ended
// ones too) is a deactivated user, never his id; a member, by his display name. Only a member is one to link to.

const members = vi.hoisted(() => new Map<string, Pick<MemberUser, "id" | "display_name">>());
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: (id: string) => members.get(id) }) }));

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  members.clear();
  members.set("u-ann", { id: "u-ann", display_name: "ann" });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

function Actor({ id }: { id: string | null }) {
  const { name, member } = useActorName(id);
  return <p data-member={String(member)}>{name}</p>;
}

/** The name the hook gives the actor, and whether he is a member. */
async function shown(id: string | null) {
  await act(async () => root.render(<Actor id={id} />));
  const actor = host.querySelector("p")!;
  return { name: actor.textContent, member: actor.dataset.member };
}

it("names a member by his display name", async () => {
  expect(await shown("u-ann")).toEqual({ name: "ann", member: "true" });
});

it("names no account nerve itself, by the site's name", async () => {
  expect(await shown(null)).toEqual({ name: "Nerve", member: "false" });
});

it("names an account the members do not list a deactivated user, never by its id", async () => {
  expect(await shown("u-gone")).toEqual({ name: "common.deactivated_user", member: "false" });
});
