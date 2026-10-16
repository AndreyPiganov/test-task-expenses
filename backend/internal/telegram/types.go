package telegram

type Update struct {
	ID      int      `json:"update_id"`
	Message *Message `json:"message,omitempty"`
}

type Message struct {
	ID   int    `json:"message_id"`
	From *User  `json:"from,omitempty"`
	Chat Chat   `json:"chat"`
	Text string `json:"text,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID int64 `json:"id"`
}
