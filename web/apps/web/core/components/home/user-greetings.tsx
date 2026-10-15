/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve types
import { useTranslation } from "@nerve/i18n";
import type { User } from "@nerve/api-client";
// hooks
import { useCurrentTime } from "@/hooks/use-current-time";

export interface IUserGreetingsView {
  user: User;
}

export function UserGreetingsView(props: IUserGreetingsView) {
  const { user } = props;
  // current time hook
  const { currentTime } = useCurrentTime();
  // store hooks
  const { t } = useTranslation();

  // the greeting, the date, the weekday and the clock in one zone, the account's; the browser's when it has none
  const inZone = (options: Intl.DateTimeFormatOptions) =>
    new Intl.DateTimeFormat("en-US", { ...options, timeZone: user.user_timezone || undefined }).format(currentTime);

  const hour = inZone({ hour12: false, hour: "numeric" });
  const date = inZone({ month: "short", day: "numeric" });
  const weekDay = inZone({ weekday: "long" });
  const timeString = inZone({ hour12: false, hour: "2-digit", minute: "2-digit" });

  const greeting = parseInt(hour, 10) < 12 ? "morning" : parseInt(hour, 10) < 18 ? "afternoon" : "evening";

  return (
    <div className="my-6 flex flex-col items-center">
      <h2 className="text-center text-20 font-semibold">{t(`greeting.${greeting}`, { name: user.display_name })}</h2>
      <h5 className="flex items-center gap-2 font-medium text-placeholder">
        <div>{greeting === "morning" ? "🌤️" : greeting === "afternoon" ? "🌥️" : "🌙️"}</div>
        <div>
          {weekDay}, {date} {timeString}
        </div>
      </h5>
    </div>
  );
}
