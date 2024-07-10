package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mmModel "github.com/mattermost/mattermost/server/public/model"
)

func (a *API) registerStatisticsRoutes(r *mux.Router) {
	// statistics
	r.HandleFunc("/statistics", a.sessionRequired(a.handleStatistics)).Methods("GET")
}

func (a *API) handleStatistics(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /statistics handleStatistics
	//
	// Fetches the statistic  of the server.
	//
	// ---
	// produces:
	// - application/json
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//         "$ref": "#/definitions/BoardStatistics"
	//   default:
	//     description: internal error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	// user must have right to access analytics
	userID := getUserID(r)
	if !a.permissions.HasPermissionTo(userID, mmModel.PermissionGetAnalytics) {
		a.errorResponse(w, r, model.NewErrPermission("access denied System Statistics"))
		return
	}

	boardCount, err := a.app.GetBoardCount()
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}
	cardCount, err := a.app.GetUsedCardsCount()
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	stats := model.BoardsStatistics{
		Boards: int(boardCount),
		Cards:  cardCount,
	}
	data, err := json.Marshal(stats)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)
}
