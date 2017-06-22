FROM thbono/go-oracle:1.0

COPY src /go/src

RUN go get -d -v github.com/streadway/amqp

RUN go install worker

ENTRYPOINT ["/go/bin/worker"]