/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";

/**
 * What the page says when it cannot load the instance's settings (InstanceWrapper): nerve did not answer, or
 * answered with a failure, maybe only for a moment. The request is tried again by itself (SWR), and the page
 * carries on once it succeeds.
 */
export function MaintenanceMessage() {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2.5">
      <h1 className="text-left text-18 font-semibold text-primary">{t("maintenance.title")}</h1>
      <span className="text-left text-14 font-medium text-secondary">{t("maintenance.description")}</span>
    </div>
  );
}
