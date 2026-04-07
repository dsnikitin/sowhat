package gigachat

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	FunctionCall string    `json:"auto"`
}

func NewRequest(msgs []Message) *Request {
	return &Request{
		Model:        "GigaChat",
		Messages:     msgs,
		FunctionCall: "auto",
	}
}
