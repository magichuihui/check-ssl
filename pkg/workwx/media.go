package workwx

import (
    "bytes"
    "io"
    "mime/multipart"
    "os"
)

const mediaFieldName = "media"

type Media struct {
    filename string
    filesize int64
    stream   io.Reader
}

func NewMediaFromFile(f *os.File) (*Media, error) {
    stat, err := f.Stat()
    if err != nil {
        return nil, err
    }

    return &Media{
        filename: stat.Name(),
        filesize: stat.Size(),
        stream:   f,
    }, nil
}

func NewMediaFromBuffer(filename string, buf []byte) (*Media, error) {
    stream := bytes.NewReader(buf)
    return &Media{
        filename: filename,
        filesize: int64(len(buf)),
        stream:   stream,
    }, nil
}

func (m *Media) writeTo(w *multipart.Writer) error {
    wr, err := w.CreateFormFile(mediaFieldName, m.filename)
    if err != nil {
        return err
    }

    _, err = io.Copy(wr, m.stream)
    if err != nil {
        return err
    }

    return nil
}
