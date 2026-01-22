// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	FigmaAPIBaseURL = "https://api.figma.com"
	MaxNodeSize     = 2000
)

type FigmaPreviewRequest struct {
	FileKey string `json:"fileKey"`
	NodeID  string `json:"nodeId"`
	BoardID string `json:"boardId"`
}

type FigmaPreviewResponse struct {
	FileID string `json:"fileId"`
	Error  string `json:"error,omitempty"`
}

type FigmaNodeResponse struct {
	Nodes map[string]struct {
		Document struct {
			AbsoluteRenderBounds struct {
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"absoluteRenderBounds"`
		} `json:"document"`
	} `json:"nodes"`
}

type FigmaImageResponse struct {
	Images map[string]string `json:"images"`
}

func (a *API) registerFigmaRoutes(r *mux.Router) {
	// Figma integration APIs
	r.HandleFunc("/figma/preview", a.sessionRequired(a.handleFigmaPreview)).Methods("POST")
}

func (a *API) handleFigmaPreview(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /figma/preview figmaPreview
	//
	// Generate a preview image from a Figma node
	//
	// ---
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     type: object
	//     properties:
	//       fileKey:
	//         type: string
	//       nodeId:
	//         type: string
	//       boardId:
	//         type: string
	// security:
	// - BearerAuth: []
	// responses:
	//   '200':
	//     description: success
	//     schema:
	//       type: object
	//       properties:
	//         fileId:
	//           type: string
	//   '400':
	//     description: bad request
	//   '403':
	//     description: forbidden
	//   '500':
	//     description: internal error

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	var req FigmaPreviewRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		a.errorResponse(w, r, model.NewErrBadRequest(err.Error()))
		return
	}

	if req.FileKey == "" || req.NodeID == "" || req.BoardID == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("fileKey, nodeId, and boardId are required"))
		return
	}

	auditRec := a.makeAuditRecord(r, "figmaPreview", audit.Fail)
	defer a.audit.LogRecord(audit.LevelModify, auditRec)
	auditRec.AddMeta("fileKey", req.FileKey)
	auditRec.AddMeta("nodeId", req.NodeID)
	auditRec.AddMeta("boardId", req.BoardID)

	// Get Figma token from configuration
	figmaToken := a.app.GetFigmaToken()
	if figmaToken == "" {
		a.errorResponse(w, r, model.NewErrBadRequest("Figma Personal Access Token not configured"))
		return
	}

	// Convert node ID from URL format (261-10355) to API format (261:10355)
	apiNodeID := strings.ReplaceAll(req.NodeID, "-", ":")

	// Check node dimensions
	nodeURL := fmt.Sprintf("%s/v1/files/%s/nodes?ids=%s", FigmaAPIBaseURL, req.FileKey, apiNodeID)
	nodeReq, err := http.NewRequest("GET", nodeURL, nil)
	if err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to create request: %w", err))
		return
	}
	nodeReq.Header.Set("X-Figma-Token", figmaToken)

	client := &http.Client{}
	nodeResp, err := client.Do(nodeReq)
	if err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to fetch node info: %w", err))
		return
	}
	defer nodeResp.Body.Close()

	if nodeResp.StatusCode != 200 {
		body, _ := io.ReadAll(nodeResp.Body)
		a.errorResponse(w, r, fmt.Errorf("figma API error: %s", string(body)))
		return
	}

	// Parse node response to check dimensions
	var nodeData FigmaNodeResponse
	if err := json.NewDecoder(nodeResp.Body).Decode(&nodeData); err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to parse node data: %w", err))
		return
	}

	// Check if node exists and get dimensions
	nodeInfo, exists := nodeData.Nodes[apiNodeID]
	if !exists {
		a.errorResponse(w, r, model.NewErrBadRequest("Node not found in Figma file"))
		return
	}

	width := nodeInfo.Document.AbsoluteRenderBounds.Width
	height := nodeInfo.Document.AbsoluteRenderBounds.Height

	if width > MaxNodeSize || height > MaxNodeSize {
		a.errorResponse(w, r, model.NewErrBadRequest(fmt.Sprintf("Node too large: %.0fx%.0f pixels (max %dx%d)", width, height, MaxNodeSize, MaxNodeSize)))
		return
	}

	// Get image URL from Figma
	imageURL := fmt.Sprintf("%s/v1/images/%s?ids=%s&format=png", FigmaAPIBaseURL, req.FileKey, apiNodeID)
	imageReq, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to create image request: %w", err))
		return
	}
	imageReq.Header.Set("X-Figma-Token", figmaToken)

	imageResp, err := client.Do(imageReq)
	if err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to get image URL: %w", err))
		return
	}
	defer imageResp.Body.Close()

	if imageResp.StatusCode != 200 {
		body, _ := io.ReadAll(imageResp.Body)
		a.errorResponse(w, r, fmt.Errorf("figma image API error: %s", string(body)))
		return
	}

	var imageData FigmaImageResponse
	if err := json.NewDecoder(imageResp.Body).Decode(&imageData); err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to parse image data: %w", err))
		return
	}

	downloadURL, exists := imageData.Images[apiNodeID]
	if !exists || downloadURL == "" {
		a.errorResponse(w, r, fmt.Errorf("failed to get image download URL"))
		return
	}

	// Download the image
	downloadResp, err := http.Get(downloadURL)
	if err != nil {
		a.errorResponse(w, r, fmt.Errorf("failed to download image: %w", err))
		return
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != 200 {
		a.errorResponse(w, r, fmt.Errorf("failed to download image from Figma"))
		return
	}

	// Get board to determine team ID
	board, err := a.app.GetBoard(req.BoardID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	// Save the image file
	filename := fmt.Sprintf("figma-%s-%s.png", req.FileKey, req.NodeID)
	fileID, err := a.app.SaveFile(downloadResp.Body, board.TeamID, req.BoardID, filename, false)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	a.logger.Debug("Figma preview generated",
		mlog.String("fileKey", req.FileKey),
		mlog.String("nodeId", req.NodeID),
		mlog.String("fileID", fileID),
	)

	response := FigmaPreviewResponse{
		FileID: fileID,
	}

	data, err := json.Marshal(response)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	jsonBytesResponse(w, http.StatusOK, data)

	auditRec.AddMeta("fileID", fileID)
	auditRec.Success()
}
