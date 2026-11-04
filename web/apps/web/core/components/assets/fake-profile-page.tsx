/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import type { User, UserUpdate } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import { json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { accountOf } from "@/store/member/fake-users";
import { profileOf, seedAccount } from "@/store/user/fake-account";
import { pickerTab, waitForPickerDOM } from "./fake-image-picker";

/** A real page's account, image service and store. Only network responses and browser APIs are controlled. */
export async function profilePage() {
  const tab = await pickerTab();
  let account = accountOf("Ada");
  await seedAccount(tab.nerve, tab.app.user, account);
  const accept = (call: Call) => {
    const changes = call.body as UserUpdate;
    account = { ...account, ...changes };
    for (const [id, url, expires] of [
      ["avatar_asset_id", "avatar_url", "avatar_url_expires_at"],
      ["cover_image_asset_id", "cover_image_url", "cover_image_url_expires_at"],
    ] as const) {
      if (id in changes) {
        account[url] = changes[id] ? `/static/${changes[id]}` : null;
        account[expires] = changes[id] ? new Date(Date.now() + 60_000).toISOString() : null;
      }
    }
    call.answer(json(200, account));
  };
  const answer = (call: Call) => {
    if (call.path === "/api/v0/me") {
      if (call.method === "PATCH") accept(call);
      else call.answer(json(200, account));
    } else if (call.path === "/api/v0/me/profile") call.answer(json(200, profileOf()));
    else tab.answer(call);
  };
  tab.nerve.respond = answer;
  const config = { provider: () => new Map(), revalidateOnMount: true };
  const render = (children: ReactNode) =>
    act(async () =>
      tab.renderer.render(
        <SWRConfig value={config}>
          <StoreContext.Provider value={tab.app}>
            <MemoryRouter>{children}</MemoryRouter>
          </StoreContext.Provider>
        </SWRConfig>
      )
    );
  const closed = () => waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "saved image dialog");
  const uploaded = async (ordinal = tab.puts.length + 1) => {
    await tab.select();
    await tab.click("image_upload.upload_save");
    await act(async () => until(() => tab.puts.length === ordinal, "raw image PUT"));
    await act(async () => tab.puts[ordinal - 1]!.answer(new Response(null, { status: 204 })));
  };
  return {
    ...tab,
    render,
    closed,
    uploaded,
    answer,
    accept,
    writes: () => tab.nerve.calls.filter((call) => call.path === "/api/v0/me" && call.method === "PATCH"),
    account: (): User => account,
  };
}
