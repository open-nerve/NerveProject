/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { Command } from "cmdk";
import { observer } from "mobx-react";
// nerve imports
import { ISSUE_PRIORITIES } from "@nerve/constants";
import { PriorityIcon } from "@nerve/propel/icons";
import type { TIssuePriorities } from "@nerve/types";
// local imports
import { PowerKModalCommandItem } from "../../../modal/command-item";

type Props = {
  handleSelect: (priority: TIssuePriorities) => void;
  workItemDetails: Issue;
};

export const PowerKWorkItemPrioritiesMenu = observer(function PowerKWorkItemPrioritiesMenu(props: Props) {
  const { handleSelect, workItemDetails } = props;

  return (
    <Command.Group>
      {ISSUE_PRIORITIES.map((priority) => (
        <PowerKModalCommandItem
          key={priority.key}
          iconNode={<PriorityIcon priority={priority.key} />}
          label={priority.title}
          isSelected={priority.key === workItemDetails.priority}
          onSelect={() => handleSelect(priority.key)}
        />
      ))}
    </Command.Group>
  );
});
