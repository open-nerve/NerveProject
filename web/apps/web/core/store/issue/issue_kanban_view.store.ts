/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable } from "mobx";
import { computedFn } from "mobx-utils";
import { DRAG_ALLOWED_GROUPS } from "@nerve/constants";
// types
import type { TIssueGroupByOptions } from "@nerve/types";

export interface IIssueKanBanViewStore {
  isDragging: boolean;
  // computed
  getCanUserDragDrop: (
    group_by: TIssueGroupByOptions | undefined,
    sub_group_by: TIssueGroupByOptions | undefined
  ) => boolean;
  // actions
  beginDrag: () => () => void;
}

export class IssueKanBanViewStore implements IIssueKanBanViewStore {
  isDragging = false;
  private dragOwner: symbol | undefined;
  constructor() {
    makeObservable(this, {
      isDragging: observable.ref,
      // actions
      beginDrag: action.bound,
    });
  }

  beginDrag = () => {
    const owner = Symbol("kanban drag");
    this.dragOwner = owner;
    this.isDragging = true;
    return action(() => {
      if (this.dragOwner !== owner) return;
      this.dragOwner = undefined;
      this.isDragging = false;
    });
  };

  getCanUserDragDrop = computedFn(
    (group_by: TIssueGroupByOptions | undefined, sub_group_by: TIssueGroupByOptions | undefined) => {
      if (group_by && DRAG_ALLOWED_GROUPS.includes(group_by)) {
        if (!sub_group_by) return true;
        if (sub_group_by && DRAG_ALLOWED_GROUPS.includes(sub_group_by)) return true;
      }
      return false;
    }
  );
}
