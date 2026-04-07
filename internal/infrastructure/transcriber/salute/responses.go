package salute

import "github.com/google/uuid"

type UploadResponse struct {
	Status int `json:"status"`
	Result struct {
		FileId uuid.UUID `json:"request_file_id"`
	} `json:"result"`
}

type RecognizeResponse struct {
	Status int `json:"status"`
	Result struct {
		TaksId string `json:"id"`
	} `json:"result"`
}

type CheckTaskResponse struct {
	Result struct {
		Status         string `json:"status"`
		ResponseFileID string `json:"response_file_id"`
	} `json:"result"`
}

type DownloadResponse []struct {
	Results []struct {
		Text string `json:"text"`
	} `json:"results"`
}
