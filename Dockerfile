FROM technology-cloud-platform.tencentcloudcr.com/public/golang:1.18 as builder

COPY . /app

WORKDIR /app/cmd

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o check-ssl main.go


FROM technology-cloud-platform.tencentcloudcr.com/public/alpine:3.15

COPY --from=builder /app/cmd/check-ssl /check-ssl
COPY config.yaml /config.yaml

CMD ["/check-ssl"]
