/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { StateGroupIcon } from "@nerve/propel/icons";
import type { StateGroup } from "@nerve/api-client";
// components
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";

export type TStateGroupData = {
  state: string | undefined;
  completed: number;
}[];

type TStateGroupStatComponent = {
  distribution: TStateGroupData;
  totalIssuesCount: number;
};

export const StateGroupStatComponent = observer(function StateGroupStatComponent(props: TStateGroupStatComponent) {
  const { distribution, totalIssuesCount } = props;

  return (
    <div>
      {distribution.map((group) => (
        <SingleProgressStats
          key={group.state}
          title={
            <div className="flex items-center gap-2">
              <StateGroupIcon stateGroup={group.state as StateGroup} />
              <span className="text-11 capitalize">{group.state}</span>
            </div>
          }
          completed={group.completed}
          total={totalIssuesCount}
        />
      ))}
    </div>
  );
});
