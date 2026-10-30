/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
// nerve imports
import type { IFilterInstance } from "@nerve/shared-state";
import type {
  SingleOrArray,
  TExternalFilter,
  TFilterProperty,
  TFilterValue,
  TFilterConditionNode,
  TSupportedOperators,
} from "@nerve/types";
import { CustomSearchSelect } from "@nerve/ui";
import { cn } from "@nerve/utils";
// local imports
import { FilterValueInput } from "../filter-value-input/root";
import { COMMON_FILTER_ITEM_BORDER_CLASSNAME } from "../shared";
import { FilterItemCloseButton } from "./close-button";
import { FilterItemContainer } from "./container";
import { InvalidFilterItem } from "./invalid";
import { FilterItemLoader } from "./loader";
import { FilterItemProperty } from "./property";

export interface IFilterItemProps<P extends TFilterProperty, E extends TExternalFilter> {
  condition: TFilterConditionNode<P, TFilterValue>;
  filter: IFilterInstance<P, E>;
  isDisabled?: boolean;
  showTransition?: boolean;
}

export const FilterItem = observer(function FilterItem<P extends TFilterProperty, E extends TExternalFilter>(
  props: IFilterItemProps<P, E>
) {
  const { condition, filter, isDisabled = false, showTransition = true } = props;
  const { t } = useTranslation();
  // derived values
  const filterConfig = condition?.property ? filter.configManager.getConfigByProperty(condition.property) : undefined;
  const operatorLabel = (operator: TSupportedOperators | undefined) => {
    if (!operator) return "--";
    const custom = filterConfig?.getOperatorConfig(operator)?.operatorLabel;
    if (custom) return custom;
    const keys: Record<TSupportedOperators, string> = {
      exact: "common.filter_options.operators.exact",
      in: "common.filter_options.operators.in",
      range: "common.filter_options.operators.range",
    };
    return t(keys[operator]);
  };
  const operatorOptions = filterConfig?.getAllDisplayOperatorOptionsByValue(condition.value).map((option) => ({
    value: option.value,
    content: operatorLabel(filterConfig.getDisplayOperatorByValue(option.value, condition.value)),
    query: operatorLabel(filterConfig.getDisplayOperatorByValue(option.value, condition.value)).toLowerCase(),
  }));
  const selectedOperatorFieldConfig = filterConfig?.getOperatorConfig(condition.operator);
  const selectedOperatorOption = filterConfig?.getDisplayOperatorByValue(condition.operator, condition.value);
  // Disable operator selection when filter is disabled or only one operator option is available and selected
  const isOperatorSelectionDisabled =
    isDisabled ||
    (condition.operator && operatorOptions?.length === 1 && operatorOptions[0]?.value === condition.operator);

  const handleOperatorChange = (operator: TSupportedOperators) => {
    if (operator) {
      filter.updateConditionOperator(condition.id, operator);
    }
  };

  const handleValueChange = (values: SingleOrArray<TFilterValue>) => {
    filter.updateConditionValue(condition.id, values);
  };

  if (!filter.configManager.areConfigsReady) {
    return <FilterItemLoader />;
  }

  if (!filterConfig) {
    return (
      <InvalidFilterItem
        condition={condition}
        filter={filter}
        isDisabled={isDisabled}
        showTransition={showTransition}
      />
    );
  }

  return (
    <FilterItemContainer conditionValue={condition.value} showTransition={showTransition}>
      {/* Property section */}
      <FilterItemProperty
        conditionId={condition.id}
        filter={filter}
        icon={filterConfig.icon}
        isDisabled={isDisabled}
        label={filterConfig.label}
        tooltipContent={typeof filterConfig.tooltipContent === "string" ? filterConfig.tooltipContent : undefined}
      />

      {/* Operator section */}
      <CustomSearchSelect
        searchPlaceholder={t("common.filter_options.search")}
        noResultsMessage={t("common.filter_options.no_results")}
        loadingMessage={t("common.filter_options.loading")}
        value={condition.operator}
        onChange={handleOperatorChange}
        options={operatorOptions}
        className={COMMON_FILTER_ITEM_BORDER_CLASSNAME}
        customButtonClassName={cn(
          "h-full px-2 text-13 font-regular",
          isOperatorSelectionDisabled && "hover:bg-layer-2-hover"
        )}
        optionsClassName="w-48"
        maxHeight="2xl"
        disabled={isOperatorSelectionDisabled}
        customButton={
          <div className="flex h-full items-center" aria-disabled={isOperatorSelectionDisabled}>
            {operatorLabel(selectedOperatorOption)}
          </div>
        }
      />

      {/* Value section */}
      {selectedOperatorFieldConfig && (
        <FilterValueInput
          filterFieldConfig={selectedOperatorFieldConfig}
          condition={condition}
          onChange={handleValueChange}
          isDisabled={isDisabled}
        />
      )}

      {/* Remove button */}
      {!isDisabled && <FilterItemCloseButton conditionId={condition.id} filter={filter} />}
    </FilterItemContainer>
  );
});
