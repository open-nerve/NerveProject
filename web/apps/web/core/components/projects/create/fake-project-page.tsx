/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import type { ProjectCreate } from "@nerve/api-client";
import { profilePage } from "@/components/assets/fake-profile-page";
import { json, type Call } from "@/lib/auth/fake-nerve";
import { projectOf } from "@/store/project/fake-projects";

export async function projectPage() {
  const page = await profilePage();
  const creations = () => page.nerve.calls.filter((c) => c.method === "POST" && c.path.endsWith("/projects"));
  const answer = (call: Call) => {
    if (call.method === "POST" && call.path.endsWith("/projects")) {
      const data = call.body as ProjectCreate;
      const project = projectOf(data.identifier, page.workspace.id, {
        ...data,
        member_role: 20,
        cover_image_asset_id: data.cover_image_asset_id ?? null,
        cover_builtin_id: data.cover_builtin_id ?? null,
        cover_image_url: data.cover_image_asset_id ? `/static/${data.cover_image_asset_id}` : null,
        cover_image_url_expires_at: data.cover_image_asset_id ? new Date(Date.now() + 60_000).toISOString() : null,
      });
      call.answer(json(201, project));
    } else page.answer(call);
  };
  page.nerve.respond = answer;
  const name = async (value = "Fresh") => {
    const input = document.querySelector<HTMLInputElement>('input[name="name"]');
    if (!input) throw new Error("project name input absent");
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  };
  const submit = async () => {
    const form = document.querySelector("form");
    if (!form) throw new Error("project form absent");
    await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  };
  return { ...page, answer, creations, name, submit };
}
