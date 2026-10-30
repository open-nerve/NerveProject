/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { set } from "lodash-es";
import { action, computed, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type {
  TSupportedOperators,
  TFilterConfig,
  TFilterProperty,
  TFilterValue,
  SingleOrArray,
  TOperatorSpecificConfigs,
} from "@nerve/types";
import { FILTER_FIELD_TYPE } from "@nerve/types";

export interface IFilterConfig<P extends TFilterProperty> extends TFilterConfig<P> {
  // computed
  allEnabledSupportedOperators: TSupportedOperators[];
  firstOperator: TSupportedOperators | undefined;
  // computed functions
  getOperatorConfig: (
    operator: TSupportedOperators
  ) => TOperatorSpecificConfigs[keyof TOperatorSpecificConfigs] | undefined;
  getDisplayOperatorByValue: <T extends TSupportedOperators>(operator: T, value: SingleOrArray<TFilterValue>) => T;
  // actions
  mutate: (updates: Partial<TFilterConfig<P>>) => void;
}

export class FilterConfig<P extends TFilterProperty> implements IFilterConfig<P> {
  // observables
  id: IFilterConfig<P>["id"];
  label: IFilterConfig<P>["label"];
  icon?: IFilterConfig<P>["icon"];
  isEnabled: IFilterConfig<P>["isEnabled"];
  supportedOperatorConfigsMap: IFilterConfig<P>["supportedOperatorConfigsMap"];
  allowMultipleFilters: IFilterConfig<P>["allowMultipleFilters"];

  /**
   * Creates a new FilterConfig instance.
   * @param params - The parameters for the filter config.
   */
  constructor(params: TFilterConfig<P>) {
    this.id = params.id;
    this.label = params.label;
    this.icon = params.icon;
    this.isEnabled = params.isEnabled;
    this.supportedOperatorConfigsMap = params.supportedOperatorConfigsMap;
    this.allowMultipleFilters = params.allowMultipleFilters;

    makeObservable(this, {
      id: observable,
      label: observable,
      icon: observable,
      isEnabled: observable,
      supportedOperatorConfigsMap: observable,
      allowMultipleFilters: observable,
      // computed
      allEnabledSupportedOperators: computed,
      firstOperator: computed,
      // actions
      mutate: action,
    });
  }

  // ------------ computed ------------

  /**
   * Returns all supported operators.
   * @returns All supported operators.
   */
  get allEnabledSupportedOperators(): IFilterConfig<P>["allEnabledSupportedOperators"] {
    return Array.from(this.supportedOperatorConfigsMap.entries())
      .filter(([, operatorConfig]) => operatorConfig.isOperatorEnabled)
      .map(([operator]) => operator);
  }

  /**
   * Returns the first operator.
   * @returns The first operator.
   */
  get firstOperator(): IFilterConfig<P>["firstOperator"] {
    return this.allEnabledSupportedOperators[0];
  }

  // ------------ computed functions ------------

  /**
   * Returns the operator config.
   * @param operator - The operator.
   * @returns The operator config.
   */
  getOperatorConfig: IFilterConfig<P>["getOperatorConfig"] = computedFn((operator) =>
    this.supportedOperatorConfigsMap.get(operator)
  );

  /**
   * Returns the operator for a value.
   * @param value - The value.
   * @returns The operator for the value.
   */
  getDisplayOperatorByValue: IFilterConfig<P>["getDisplayOperatorByValue"] = computedFn((operator, value) => {
    const operatorConfig = this.getOperatorConfig(operator);
    if (operatorConfig?.type === FILTER_FIELD_TYPE.MULTI_SELECT && (Array.isArray(value) ? value.length : 0) <= 1) {
      return operatorConfig.singleValueOperator as typeof operator;
    }
    return operator;
  });

  // ------------ actions ------------

  /**
   * Mutates the config.
   * @param updates - The updates to apply to the config.
   */
  mutate: IFilterConfig<P>["mutate"] = action((updates) => {
    runInAction(() => {
      for (const key in updates) {
        if (updates.hasOwnProperty(key)) {
          const configKey = key as keyof TFilterConfig<P>;
          set(this, configKey, updates[configKey]);
        }
      }
    });
  });
}
