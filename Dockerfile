FROM golang:alpine AS builder
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY . .
RUN go build .

FROM scratch
COPY --from=builder /app/tgPrivacyBot /tgPrivacyBot
ENV TOKEN ""
ENV USE_MYSQL "no"
ENV MYSQL_CONFIG "user:name@tcp(ip:port)/tgPrivacyBot?charset=utf8mb4&parseTime=True&loc=Local"
ENV https_proxy ""
ENV http_proxy ""
ENV all_proxy ""
ENV SEND_TO_GROUP_ID ""
ENV CRONTAB ""
ENTRYPOINT ["/tgPrivacyBot"]