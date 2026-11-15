/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { MobileHeaderFilters } from "@/components/issues/mobile-header-filters";
import { usePlanningPageHeader } from "./use-planning-page-header";

type Props = {
  /** The page's store type: a cycle's page or a module's. */
  storeType: PlanningPageStoreType;
  /** The page's address, as the route gives it. */
  workspaceSlug: string;
  projectId: string;
  /** The page's cycle or module. */
  entityId: string;
};

/**
 * The small screen's header of a cycle's or a module's page (P7c spec 2.7, R8, A8): the page's layouts and display
 * (MobileHeaderFilters, of the page's own preferences and display options) when the page's header at a wider width
 * gives its controls (usePlanningPageHeader): for those whom the page lists its work items, once nerve has given the
 * page's preferences, not before its first answer nor while that first read failed. An observer, so that an answer
 * after the page opened shows them: the layout that passes it is not one (re-76 F1).
 */
export const PlanningPageMobileHeader = observer(function PlanningPageMobileHeader(props: Props) {
  const { storeType, workspaceSlug, projectId, entityId } = props;
  const { preferencesShown } = usePlanningPageHeader(storeType, { workspaceSlug, projectId, entityId });

  if (!preferencesShown) return null;

  return <MobileHeaderFilters storeType={storeType} entityId={entityId} />;
});
