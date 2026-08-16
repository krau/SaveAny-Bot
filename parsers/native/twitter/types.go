package twitter

type FxTwitterApiResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Tweet   struct {
		URL    string `json:"url"`
		ID     string `json:"id"`
		Text   string `json:"text"`
		Author struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			ScreenName string `json:"screen_name"`
			Protected  bool   `json:"protected"`
		} `json:"author"`
		PossiblySensitive bool   `json:"possibly_sensitive"`
		IsNoteTweet       bool   `json:"is_note_tweet"`
		Lang              string `json:"lang"`
		Media             struct {
			All []struct {
				URL  string `json:"url"`
				Type string `json:"type"`
			} `json:"all"`
		} `json:"media"`
	} `json:"tweet"`
}
