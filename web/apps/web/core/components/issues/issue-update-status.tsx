/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";
import { observer } from "mobx-react";
import { RefreshOutline } from "@makeplane/propel/icons";
// types
import type { TNameDescriptionLoader } from "@nerve/types";

type Props = {
  isSubmitting: TNameDescriptionLoader;
};

export const NameDescriptionUpdateStatus = observer(function NameDescriptionUpdateStatus(props: Props) {
  const { isSubmitting } = props;
  const { t } = useTranslation();

  return (
    <>
      <div
        className={`flex items-center gap-x-2 transition-all duration-300 ${
          isSubmitting === "saved" ? "fade-out" : "fade-in"
        }`}
      >
        {isSubmitting === "submitting" && <RefreshOutline className="size-3.5 animate-spin text-tertiary" />}
        <span className="text-13 text-tertiary">
          {t(
            isSubmitting === "submitting"
              ? "common.saving"
              : isSubmitting === "failed"
                ? "common.save_failed"
                : "common.saved"
          )}
        </span>
      </div>
    </>
  );
});
