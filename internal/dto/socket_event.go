package dto

// BindUserOKResponse is sent after bind_user succeeds.
type BindUserOKResponse struct {
	ERP string `json:"erp"`
}

// BindUserPayload is the incoming payload of bind_user.
type BindUserPayload struct {
	ERP   string `json:"erp"`
	Token string `json:"token"`
}

// SendToUserPayload is the incoming payload of send_to_user.
type SendToUserPayload struct {
	ToERP   string `json:"toErp"`
	Message string `json:"message"`
}

// SendToUserOKResponse is sent after send_to_user succeeds.
type SendToUserOKResponse struct {
	ToERP string `json:"toErp"`
}

// PrivateMessagePayload is the outgoing payload of private_message.
type PrivateMessagePayload struct {
	Message string `json:"message"`
}

// CrossNodePrivateMessage is used for redis pub/sub fanout between nodes.
type CrossNodePrivateMessage struct {
	Event          string `json:"event"`
	ToERP          string `json:"toErp"`
	FromERP        string `json:"fromErp,omitempty"`
	ConversationID uint   `json:"conversationId,omitempty"`
	MessageID      uint   `json:"messageId,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
	Message        string `json:"message"`
}

// SendToGroupPayload is the incoming payload of send_to_group.
type SendToGroupPayload struct {
	ConversationID uint   `json:"conversationId"`
	Message        string `json:"message"`
	ClientMsgID    string `json:"clientMsgId"`
}

// GroupMessagePayload is the outgoing payload of group_message.
type GroupMessagePayload struct {
	ConversationID uint   `json:"conversationId"`
	FromERP        string `json:"fromErp"`
	MessageID      uint   `json:"messageId"`
	Message        string `json:"message"`
	CreatedAt      string `json:"createdAt"`
}

// SendToGroupOKResponse is sent after send_to_group succeeds.
type SendToGroupOKResponse struct {
	ConversationID uint `json:"conversationId"`
	MessageID      uint `json:"messageId"`
	RecipientCount int  `json:"recipientCount"`
	Deduplicated   bool `json:"deduplicated"`
}
