/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// local imports
import type { IntakeList as IntakeListRead } from "../use-intake-list";
import { IntakeList } from "./list";
import { IntakeTabs } from "./tabs";

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The list of the tab shown, and its read (useIntakeList). */
  list: IntakeListRead;
  /** The work item the page shows. */
  selectedId: string | undefined;
  /** The user pressed a row. */
  onOpen: () => void;
};

/**
 * The intake page's sidebar (P5a spec 2.7): its tabs, open and closed, the open one saying its list's total, and the
 * chosen tab's list.
 */
export const IntakeSidebar = observer(function IntakeSidebar(props: Props) {
  return (
    <div className="h-full w-full flex-shrink-0 border-r border-strong bg-surface-1">
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <IntakeTabs total={props.list.page?.total}>
          <IntakeList {...props} />
        </IntakeTabs>
      </div>
    </div>
  );
});
