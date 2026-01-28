// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export enum RelationType {
    Blocks = 'blocks',
    BlockedBy = 'blocked_by',
    RelatesTo = 'relates_to',
    Duplicates = 'duplicates',
    DuplicatedBy = 'duplicated_by',
    ParentOf = 'parent_of',
    ChildOf = 'child_of',
}

export type CardRelation = {
    id: string
    sourceCardId: string
    targetCardId: string
    relationType: RelationType
    createAt: number
    updateAt: number
    createdBy: string
    boardId: string
}

export function createCardRelation(relation?: Partial<CardRelation>): CardRelation {
    return {
        id: relation?.id || '',
        sourceCardId: relation?.sourceCardId || '',
        targetCardId: relation?.targetCardId || '',
        relationType: relation?.relationType || RelationType.RelatesTo,
        createAt: relation?.createAt || 0,
        updateAt: relation?.updateAt || 0,
        createdBy: relation?.createdBy || '',
        boardId: relation?.boardId || '',
    }
}

export function getInverseRelationType(relationType: RelationType): RelationType {
    switch (relationType) {
    case RelationType.Blocks:
        return RelationType.BlockedBy
    case RelationType.BlockedBy:
        return RelationType.Blocks
    case RelationType.Duplicates:
        return RelationType.DuplicatedBy
    case RelationType.DuplicatedBy:
        return RelationType.Duplicates
    case RelationType.ParentOf:
        return RelationType.ChildOf
    case RelationType.ChildOf:
        return RelationType.ParentOf
    case RelationType.RelatesTo:
    default:
        return RelationType.RelatesTo
    }
}

export function getRelationTypeDisplayName(relationType: RelationType): string {
    switch (relationType) {
    case RelationType.Blocks:
        return 'Blocks'
    case RelationType.BlockedBy:
        return 'Blocked by'
    case RelationType.RelatesTo:
        return 'Relates to'
    case RelationType.Duplicates:
        return 'Duplicates'
    case RelationType.DuplicatedBy:
        return 'Duplicated by'
    case RelationType.ParentOf:
        return 'Parent of'
    case RelationType.ChildOf:
        return 'Child of'
    default:
        return 'Relates to'
    }
}

