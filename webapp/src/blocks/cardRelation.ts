// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export enum RelationType {
    Blocks = 'blocks',
    BlockedBy = 'is_blocked_by',
    RelatesTo = 'relates_to',
    Duplicates = 'duplicates',
    DuplicatedBy = 'is_duplicated_by',
    Clones = 'clones',
    ClonedBy = 'is_cloned_by',
    Causes = 'causes',
    CausedBy = 'is_caused_by',
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
    case RelationType.Clones:
        return RelationType.ClonedBy
    case RelationType.ClonedBy:
        return RelationType.Clones
    case RelationType.Causes:
        return RelationType.CausedBy
    case RelationType.CausedBy:
        return RelationType.Causes
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
    case RelationType.Clones:
        return 'Clones'
    case RelationType.ClonedBy:
        return 'Cloned by'
    case RelationType.Causes:
        return 'Causes'
    case RelationType.CausedBy:
        return 'Caused by'
    default:
        return 'Relates to'
    }
}

