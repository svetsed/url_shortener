package model

type URL struct {
    ID          string
    OriginalURL string
    ShortURL    string
    // CreatedAt   time.Time
    // ExpiresAt   *time.Time
    // ClickCount  int
}

type RequestJSON struct {
    URL string `json:"url"`
}

type ResponseJSON struct {
    Result string `json:"result"`
}