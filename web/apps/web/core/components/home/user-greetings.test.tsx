/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import type { User } from "@nerve/api-client";
import { UserGreetingsView } from "./user-greetings";

// The home's greeting (M3 closeout, C12-3): its greeting, date, weekday and clock all in the account's time zone.
// The process's zone is UTC here, whatever the machine's: the moment is 23:50 on Wednesday 14 October in UTC, and
// 07:50 on Thursday 15 October in Shanghai.

const clock = vi.hoisted(() => ({ now: new Date("2026-10-14T23:50:00Z") }));
vi.mock("@/hooks/use-current-time", () => ({ useCurrentTime: () => ({ currentTime: clock.now }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const ada: User = {
  id: "u-ada",
  email: "ada@example.com",
  first_name: "Ada",
  last_name: "Lovelace",
  display_name: "ada",
  user_timezone: "Asia/Shanghai",
  avatar_url: null,
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
};

/** The greeting's key and the line under it, as the home shows them to user. */
function greeted(user: User): { greeting: string | undefined; line: string | undefined } {
  const html = renderToStaticMarkup(<UserGreetingsView user={user} />);
  return {
    greeting: /<h2[^>]*>([^<]*)<\/h2>/.exec(html)?.[1],
    line: /<div>([^<]*, [^<]*)<\/div>/.exec(html)?.[1],
  };
}

// the process's zone, which Node reads from TZ as it changes
beforeAll(() => {
  vi.stubEnv("TZ", "UTC");
});
afterAll(() => {
  vi.unstubAllEnvs();
});

describe("UserGreetingsView", () => {
  it("greets, dates and clocks the moment in the account's zone, not the browser's", () => {
    expect(greeted(ada)).toEqual({ greeting: "greeting.morning", line: "Thursday, Oct 15 07:50" });
  });

  it("uses the browser's zone when the account has none", () => {
    expect(greeted({ ...ada, user_timezone: "" })).toEqual({
      greeting: "greeting.evening",
      line: "Wednesday, Oct 14 23:50",
    });
  });
});
