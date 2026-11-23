/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { Tab, TabGroup, TabList, TabPanel, TabPanels } from "@headlessui/react";
// nerve imports
import { INTAKE_TABS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// components
import { CountChip } from "@/components/common/count-chip";
// local imports
import { useIntakePageState } from "../page-state";

type Props = {
  /** The number of work items the list of the tab shown selects, as nerve last answered it; undefined until read. */
  total: number | undefined;
  /** What the chosen tab's panel shows: the list of its work items. */
  children: ReactNode;
};

/**
 * The intake page's tabs, open and closed (P5a spec 2.6, 2.7, R31): tabs as a screen reader reads them (Headless
 * Tab), the address's tab chosen. The open tab, chosen, says its list's total once read, its name too; the closed tab
 * never does. Choosing one goes to its address, ?currentTab alone (a push: the user's move), the selection left
 * behind: the page then selects the new list's first row. Its panel shows the children.
 */
export const IntakeTabs = observer(function IntakeTabs({ total, children }: Props) {
  const { t } = useTranslation();
  const { filters } = useIntakePageState();
  const [, setParams] = useSearchParams();
  return (
    <TabGroup
      as="div"
      className="flex min-h-0 flex-1 flex-col"
      selectedIndex={Math.max(
        0,
        INTAKE_TABS.findIndex((tab) => tab.value === filters.tab)
      )}
      onChange={(index: number) => setParams({ currentTab: INTAKE_TABS[index]?.value ?? "open" })}
    >
      <TabList className="flex h-header flex-shrink-0 items-stretch border-b border-subtle px-3">
        {INTAKE_TABS.map((tab) => {
          const label = t(tab.i18n_label);
          const shown = tab.value === "open" && filters.tab === "open" ? total : undefined;
          return (
            <Tab
              key={tab.value}
              aria-label={shown === undefined ? label : t("inbox_issue.a11y.tab_total", { label, count: shown })}
              className={({ selected }) =>
                cn(
                  "relative flex items-center gap-1 px-3 text-13 font-medium transition-all focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong",
                  selected
                    ? "text-accent-primary after:absolute after:inset-x-0 after:bottom-0 after:border-b after:border-accent-strong"
                    : "text-primary hover:text-secondary"
                )
              }
            >
              {label}
              {shown !== undefined && <CountChip count={shown} />}
            </Tab>
          );
        })}
      </TabList>
      <TabPanels className="flex min-h-0 flex-1 flex-col">
        {INTAKE_TABS.map((tab) => (
          <TabPanel key={tab.value} className="flex min-h-0 flex-1 flex-col">
            {children}
          </TabPanel>
        ))}
      </TabPanels>
    </TabGroup>
  );
});
