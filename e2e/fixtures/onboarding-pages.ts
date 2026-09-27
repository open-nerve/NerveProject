import type { Page } from "@playwright/test";

// The profile step of onboarding (M2 design 3.1), as a person takes it.

/**
 * Clicks "Continue" on the profile step that page shows, the name filled in, and resolves with the status of
 * nerve's answer to the step's last request, which saves the step done after the name. That request needs a
 * token, so every refresh the step waited for came before it.
 */
export async function saveProfileStep(page: Page): Promise<number> {
  const [saved] = await Promise.all([
    page.waitForResponse(
      (response) =>
        response.request().method() === "PATCH" && new URL(response.url()).pathname === "/api/v0/me/profile",
      { timeout: 10_000 }
    ),
    page.getByRole("button", { name: "Continue", exact: true }).click(),
  ]);
  return saved.status();
}
