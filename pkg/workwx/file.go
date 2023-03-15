package workwx

type FileMessage struct {
    Message
    File File `json:"file"`
}

type File struct {
    MediaID string `json:"media_id"`
}
