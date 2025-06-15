package alist

import "errors"

var (
	ErrAlistLoginFailed = errors.New("failed to login to Alist")
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

type meResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"data"`
}

type putResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Task struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			State    int    `json:"state"`
			Status   string `json:"status"`
			Progress int    `json:"progress"`
			Error    string `json:"error"`
		} `json:"task"`
	} `json:"data"`
}

type fsGetResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
