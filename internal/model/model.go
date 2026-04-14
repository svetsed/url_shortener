package model

type URL struct {
    ID          string `json:"uuid"`
    UserID      string `json:"user_id"`
    OriginalURL string `json:"original_url"`
    ShortURL    string `json:"short_url"`
    NeedDelete  bool   `json:"is_deleted"`
    // CreatedAt   time.Time
    // ExpiresAt   *time.Time
    // ClickCount  int
}

type OneURLRequest struct {
    URL    string `json:"url"`
}

type OneURLResponse struct {
    Result string `json:"result"`
}

type ManyURLRequest struct {
    CorrelationID string `json:"correlation_id"`
    OriginalURL   string `json:"original_url"`
}

type ManyURLResponse struct {
    CorrelationID string `json:"correlation_id"`
    ShortURL      string `json:"short_url"`
}