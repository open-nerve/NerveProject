/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { NETWORK_CHOICES, PROJECT_FILTER_I18N_LABEL } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// components
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";
// local imports
import { ProjectNetworkIcon } from "../../project-network-icon";

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (val: string) => void;
  searchQuery: string;
};

export const FilterAccess = observer(function FilterAccess(props: Props) {
  const { appliedFilters, handleUpdate, searchQuery } = props;
  // states
  const [previewEnabled, setPreviewEnabled] = useState(true);
  const { t } = useTranslation();

  const appliedFiltersCount = appliedFilters?.length ?? 0;
  // the search matches each access as the page names it
  const filteredOptions = NETWORK_CHOICES.filter((a) =>
    t(a.i18n_label).toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <>
      <FilterHeader
        title={`${t(PROJECT_FILTER_I18N_LABEL.access)}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {filteredOptions.length > 0 ? (
            filteredOptions.map((access) => (
              <FilterOption
                key={access.key}
                isChecked={!!appliedFilters?.includes(`${access.key}`)}
                onClick={() => handleUpdate(`${access.key}`)}
                icon={<ProjectNetworkIcon iconKey={access.iconKey} />}
                title={t(access.i18n_label)}
              />
            ))
          ) : (
            <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
          )}
        </div>
      )}
    </>
  );
});
