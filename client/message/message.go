package message

type Message struct {
	Content   string `json:"content"`
	Type      string `json:"type"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
}
