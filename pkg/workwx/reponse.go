package workwx

type respCommon struct {
    ErrCode int64  `json:"errcode"`
    ErrMsg  string `json:"errmsg"`
}

type respMediaUpload struct {
    respCommon

    Type      string `json:"type"`
    MediaID   string `json:"media_id"`
    CreatedAt string `json:"created_at"`
}
