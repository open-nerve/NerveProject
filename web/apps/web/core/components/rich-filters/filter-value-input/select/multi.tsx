/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
import { useTranslation } from "@nerve/i18n";
import { observer } from "mobx-react";
// nerve imports
import type { SingleOrArray, TFilterProperty, TMultiSelectFilterFieldConfig, TFilterConditionNode } from "@nerve/types";
import { CustomSearchSelect } from "@nerve/ui";
import { toFilterArray, getFilterValueLength } from "@nerve/utils";
// local imports
import { SelectedOptionsDisplay } from "./selected-options-display";
import { FilterOptionsError, getCommonCustomSearchSelectProps, getFormattedOptions } from "./shared";
import { useFilterSelectOptions } from "./use-filter-select-options";

type TMultiSelectFilterValueInputProps<P extends TFilterProperty> = {
  config: TMultiSelectFilterFieldConfig<string>;
  condition: TFilterConditionNode<P, string>;
  isDisabled?: boolean;
  onChange: (values: SingleOrArray<string>) => void;
};

export const MultiSelectFilterValueInput = observer(function MultiSelectFilterValueInput<P extends TFilterProperty>(
  props: TMultiSelectFilterValueInputProps<P>
) {
  const { config, condition, isDisabled, onChange } = props;
  const { t } = useTranslation();
  const { options, loading, error, retry } = useFilterSelectOptions(config);
  // derived values
  const formattedOptions = useMemo(() => getFormattedOptions<string>(options), [options]);

  const handleSelectChange = (values: string[]) => {
    onChange(values);
  };

  return (
    <>
      <CustomSearchSelect
        {...getCommonCustomSearchSelectProps(isDisabled)}
        value={toFilterArray(condition.value)}
        onChange={handleSelectChange}
        options={loading ? undefined : formattedOptions}
        searchPlaceholder={t("common.filter_options.search")}
        noResultsMessage={t(
          error !== undefined ? "common.filter_options.load_failed" : "common.filter_options.no_results"
        )}
        loadingMessage={t("common.filter_options.loading")}
        multiple
        disabled={isDisabled}
        customButton={<SelectedOptionsDisplay<string> selectedValue={condition.value} options={options} />}
        defaultOpen={getFilterValueLength(condition.value) === 0}
      />
      {error !== undefined && <FilterOptionsError retry={retry} disabled={isDisabled} />}
    </>
  );
});
