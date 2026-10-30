/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
import { useTranslation } from "@nerve/i18n";
import { observer } from "mobx-react";
// nerve imports
import type { TFilterProperty, TSingleSelectFilterFieldConfig, TFilterConditionNode } from "@nerve/types";
import { CustomSearchSelect } from "@nerve/ui";
import { getFilterValueLength, toFilterArray } from "@nerve/utils";
// local imports
import { SelectedOptionsDisplay } from "./selected-options-display";
import { FilterOptionsError, getCommonCustomSearchSelectProps, getFormattedOptions } from "./shared";
import { useFilterSelectOptions } from "./use-filter-select-options";

type TSingleSelectFilterValueInputProps<P extends TFilterProperty> = {
  config: TSingleSelectFilterFieldConfig<string>;
  condition: TFilterConditionNode<P, string>;
  isDisabled?: boolean;
  onChange: (value: string | null) => void;
};

export const SingleSelectFilterValueInput = observer(function SingleSelectFilterValueInput<P extends TFilterProperty>(
  props: TSingleSelectFilterValueInputProps<P>
) {
  const { config, condition, onChange, isDisabled } = props;
  const { t } = useTranslation();
  const { options, loading, error, retry } = useFilterSelectOptions(config);
  // derived values
  const formattedOptions = useMemo(() => getFormattedOptions<string>(options), [options]);

  const handleSelectChange = (value: string) => {
    if (value === condition.value) {
      onChange(null);
    } else {
      onChange(value);
    }
  };

  return (
    <>
      <CustomSearchSelect
        {...getCommonCustomSearchSelectProps(isDisabled)}
        value={toFilterArray(condition.value)[0] ?? null}
        onChange={handleSelectChange}
        options={loading ? undefined : formattedOptions}
        searchPlaceholder={t("common.filter_options.search")}
        noResultsMessage={t(
          error !== undefined ? "common.filter_options.load_failed" : "common.filter_options.no_results"
        )}
        loadingMessage={t("common.filter_options.loading")}
        multiple={false}
        disabled={isDisabled}
        customButton={
          <SelectedOptionsDisplay<string> selectedValue={condition.value} options={options} displayCount={1} />
        }
        defaultOpen={getFilterValueLength(condition.value) === 0}
      />
      {error !== undefined && <FilterOptionsError retry={retry} disabled={isDisabled} />}
    </>
  );
});
