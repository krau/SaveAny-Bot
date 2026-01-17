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
	Data    struct {
		Name     string `json:"name"`
		Size     int64  `json:"size"`
		IsDir    bool   `json:"is_dir"`
		Modified string `json:"modified"`
		Created  string `json:"created"`
		Sign     string `json:"sign"`
		Thumb    string `json:"thumb"`
		Type     int    `json:"type"`
		RawURL   string `json:"raw_url"`
		Provider string `json:"provider"`
	} `json:"data"`
}

type fsListRequest struct {
	Path     string `json:"path"`
	Password string `json:"password"`
	Page     int    `json:"page"`
	PerPage  int    `json:"per_page"`
	Refresh  bool   `json:"refresh"`
}

type fsListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Content []struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			IsDir    bool   `json:"is_dir"`
			Modified string `json:"modified"`
			Created  string `json:"created"`
			Sign     string `json:"sign"`
			Thumb    string `json:"thumb"`
			Type     int    `json:"type"`
		} `json:"content"`
		Total    int64  `json:"total"`
		Readme   string `json:"readme"`
		Header   string `json:"header"`
		Write    bool   `json:"write"`
		Provider string `json:"provider"`
	} `json:"data"`
}
