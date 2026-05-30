package dto

// ApplyAccountBody defines the request payload for account application.
type ApplyAccountBody struct {
	ERP       string  `json:"erp" binding:"required"`
	Password  string  `json:"password" binding:"required"`
	Username  string  `json:"username" binding:"required"`
	Phone     string  `json:"phone" binding:"required"`
	Nickname  *string `json:"nickname,omitempty"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	Email     *string `json:"email,omitempty"`
}

// ApplyFriendRequestBody defines the request payload for creating a friend request.
type ApplyFriendRequestBody struct {
	FromERP     string `json:"fromErp"`
	ToERP       string `json:"toErp"`
	ApplyReason string `json:"applyReason"`
}

// HandleFriendRequestBody defines the request payload for accepting/rejecting a friend request.
type HandleFriendRequestBody struct {
	OperatorERP string `json:"operatorErp"` // 被请求方的ERP
	Action      string `json:"action"`      // accept / reject
	Remark      string `json:"remark"`
}
