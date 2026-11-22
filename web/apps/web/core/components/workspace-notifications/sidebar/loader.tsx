/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { range } from "lodash-es";
// nerve imports
import { useTranslation } from "@nerve/i18n";

/**
 * The notifications page's wait for nerve (P4 spec 2.6): its list's first page, its preview's project. A screen reader
 * hears that it is loading.
 */
export function NotificationsLoader() {
  const { t } = useTranslation();
  return (
    <div role="status" className="animate-pulse divide-y divide-subtle overflow-hidden">
      <span className="sr-only">{t("common.loading")}</span>
      {range(8).map((i) => (
        <div key={i} className="flex w-full items-center gap-4 p-3">
          <span className="min-h-12 min-w-12 rounded-full bg-layer-1" />
          <div className="flex w-full flex-col gap-2.5">
            <span className="h-5 w-36 rounded-xs bg-layer-1" />
            <div className="flex w-full items-center justify-between gap-2">
              <span className="h-5 w-28 rounded-xs bg-layer-1" />
              <span className="h-5 w-16 rounded-xs bg-layer-1" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
