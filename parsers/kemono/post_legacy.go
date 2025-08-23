package kemono

type PostLegacy struct {
	Props   Props    `json:"props"`
	Results []Result `json:"results"`
}

type Props struct {
	Count uint `json:"count"`
	Limit uint `json:"limit"`
}

type Result struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
