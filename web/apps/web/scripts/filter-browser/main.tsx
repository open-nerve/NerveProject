/** Browser-only compatibility fixture. It mounts production controls, never a product route. */
import React, { useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import { initPromise, setLanguage } from "@nerve/i18n";
import { COLLECTION_OPERATOR, EQUALITY_OPERATOR, FILTER_FIELD_TYPE, FILTER_NODE_TYPE } from "@nerve/types";
import type { IFilterOption, TMultiSelectFilterFieldConfig, TSingleSelectFilterFieldConfig } from "@nerve/types";
import { SingleSelectFilterValueInput } from "../../core/components/rich-filters/filter-value-input/select/single";
import { MultiSelectFilterValueInput } from "../../core/components/rich-filters/filter-value-input/select/multi";

const params = new URLSearchParams(window.location.search);
const kind = params.get("kind") === "single" ? "single" : "multi";
const language = params.get("language") === "zh-CN" ? "zh-CN" : "en";

function Fixture() {
  const [value, setValue] = useState<string | string[]>([]);
  const configs = useMemo(() => {
    const getOptions = async (): Promise<IFilterOption<string>[]> => {
      const response = await fetch("/options");
      if (!response.ok) throw new Error("Directory unavailable");
      // The fixture endpoint supplies a known typed directory; the real component owns loading and failure.
      await response.text();
      return [{ id: "ready", value: "ready", label: language === "en" ? "Ready" : "已就绪" }];
    };
    const single: TSingleSelectFilterFieldConfig<string> = { type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions };
    const multi: TMultiSelectFilterFieldConfig<string> = {
      type: FILTER_FIELD_TYPE.MULTI_SELECT,
      getOptions,
      singleValueOperator: EQUALITY_OPERATOR.EXACT,
    };
    return { single, multi };
  }, []);
  const condition = {
    id: "browser-condition",
    type: FILTER_NODE_TYPE.CONDITION,
    property: "label_id",
    operator: COLLECTION_OPERATOR.IN,
    value,
  };
  return (
    <main className="p-16">
      <h1>Production {kind} select</h1>
      <section aria-label="Value control" className="mt-8 inline-flex h-8 border">
        {kind === "single" ? (
          <SingleSelectFilterValueInput
            config={configs.single}
            condition={condition}
            onChange={(next) => setValue(next ?? [])}
          />
        ) : (
          <MultiSelectFilterValueInput config={configs.multi} condition={condition} onChange={setValue} />
        )}
      </section>
      <output aria-label="Selected value">{JSON.stringify(value)}</output>
      <button type="button">After control</button>
    </main>
  );
}

await initPromise;
await setLanguage(language);
createRoot(document.getElementById("root")!).render(<Fixture />);
