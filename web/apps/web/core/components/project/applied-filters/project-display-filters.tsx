/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// types
import { PROJECT_DISPLAY_FILTER_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CloseOutline } from "@makeplane/propel/icons";
import type { TProjectAppliedDisplayFilterKeys } from "@nerve/types";

type Props = {
  handleRemove: (key: TProjectAppliedDisplayFilterKeys) => void;
  values: TProjectAppliedDisplayFilterKeys[];
  editable: boolean | undefined;
};

export const AppliedProjectDisplayFilters = observer(function AppliedProjectDisplayFilters(props: Props) {
  const { handleRemove, values, editable } = props;
  const { t } = useTranslation();

  return (
    <>
      {values.map((key) => {
        const filterLabel = PROJECT_DISPLAY_FILTER_OPTIONS.find((s) => s.key === key)?.i18n_label;
        const label = filterLabel ? t(filterLabel) : "";
        return (
          <div key={key} className="flex items-center gap-1 rounded-sm bg-layer-1 px-1.5 py-1 text-11">
            {label}
            {editable && (
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                onClick={() => handleRemove(key)}
                aria-label={t("common.remove_item", { name: label })}
              >
                <CloseOutline height={10} width={10} />
              </button>
            )}
          </div>
        );
      })}
    </>
  );
});
