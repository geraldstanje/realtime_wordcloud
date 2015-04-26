FROM golang

# Add directories and files
ADD . /go/src/github.com/geraldstanje/realtime_wordcloud/

RUN ["go", "get", "code.google.com/p/go.net/websocket" ]

WORKDIR /go/src/github.com/geraldstanje/realtime_wordcloud

#ADD home.html /go/src/github.com/geraldstanje/realtime_wordcloud/home.html
#ADD webserver.go /go/src/github.com/geraldstanje/realtime_wordcloud/webserver.go

ENTRYPOINT go build && ./realtime_wordcloud

EXPOSE 8080