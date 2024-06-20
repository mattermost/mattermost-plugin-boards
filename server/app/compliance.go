package app

import "github.com/mattermost/mattermost-plugin-boards/server/model"

func (a *App) GetBoardsForCompliance(opts model.QueryBoardsForComplianceOptions) ([]*model.Board, bool, error) {
	return a.store.GetBoardsForCompliance(opts)
}

func (a *App) GetBoardsComplianceHistory(opts model.QueryBoardsComplianceHistoryOptions) ([]*model.BoardHistory, bool, error) {
	return a.store.GetBoardsComplianceHistory(opts)
}

func (a *App) GetBlocksComplianceHistory(opts model.QueryBlocksComplianceHistoryOptions) ([]*model.BlockHistory, bool, error) {
	return a.store.GetBlocksComplianceHistory(opts)
}
