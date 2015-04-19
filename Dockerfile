FROM golang

# Add directories and files
ADD . /go/src/github.com/geraldstanje/realtime_wordcloud/

RUN ["go", "get", "code.google.com/p/go.net/websocket" ]

WORKDIR /go/src/github.com/geraldstanje/realtime_wordcloud

ENTRYPOINT go build && ./realtime_wordcloud

EXPOSE 8080