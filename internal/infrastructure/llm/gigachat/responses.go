package gigachat

type CompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Index int `json:"index"`
	} `json:"choices"`
}

type UploadResponse struct {
	FileId string `json:"id"`
}
