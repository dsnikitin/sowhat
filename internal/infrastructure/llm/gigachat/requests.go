package gigachat

type Message struct {
	Role        string   `json:"role"`
	Content     string   `json:"content"`
	Attachments []string `json:"attachments"`
}

type Request struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	FunctionCall string    `json:"function_call"`
}

func NewRequest(msgs []Message) *Request {
	return &Request{
		Model:        "GigaChat",
		Messages:     msgs,
		FunctionCall: "auto",
	}
}
