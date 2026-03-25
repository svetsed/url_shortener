package model

type URL struct {
    ID          string `json:"uuid"`
    OriginalURL string `json:"original_url"`
    ShortURL    string `json:"short_url"`
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