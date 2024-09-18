FROM golang AS BUILD

WORKDIR /proxy

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build proxy.go

CMD ["./start.sh"]
