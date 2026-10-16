import { expect, type Page, type TestInfo } from "@playwright/test";

import { createWorkspace, inviteAndAccept, slugFor, type Api } from "./api";
import { bearer, emailFor, type AuthTokens } from "./auth";
import { modalDialog } from "./dialogs";
import { registerOnboarded } from "./settings-pages";

// The accounts and the workspace of S7, the pages in Chinese (M3 closeout), which its two files share: the workspace's
// pages, the onboarding and the invitation (s7-chinese.spec.ts), and the project's pages and the filters
// (s7-chinese-project.spec.ts).

/** Makes the language of the account of tokens Chinese, the home's tour taken already. */
export async function speakChinese(api: Api, tokens: AuthTokens): Promise<void> {
  const chinese = await api.PATCH("/api/v0/me/profile", {
    body: { language: "zh-CN", is_tour_completed: true },
    headers: bearer(tokens.access_token),
  });
  expect(chinese.response.status).toBe(200);
}

/**
 * An admin of this run of the test whose language is Chinese, the home's tour taken already, and Acme, his workspace,
 * of the size "Just myself", with bob, a member.
 */
export async function chineseAdmin(
  api: Api,
  testInfo: TestInfo
): Promise<{ admin: AuthTokens; slug: string; bobEmail: string }> {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const bobEmail = emailFor(testInfo, "bob");
  const bob = await registerOnboarded(api, bobEmail);
  await speakChinese(api, admin);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug, organization_size: "Just myself" });
  await inviteAndAccept(api, admin.access_token, slug, { email: bobEmail, token: bob.access_token }, 15);
  return { admin, slug, bobEmail };
}

/** Closes the one modal dialog page shows by its Cancel (取消). */
export async function cancel(page: Page): Promise<void> {
  await modalDialog(page).getByRole("button", { name: "取消" }).click();
  await expect(modalDialog(page)).toHaveCount(0);
}
