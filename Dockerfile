FROM golang:latest as go

RUN mkdir /build

WORKDIR /build

RUN go get -u github.com/aws/aws-sdk-go

Run go get -u github.com/kataras/golog

COPY .git /build/
COPY aws-env.go /build

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=$(git describe --tags)" -v -o awsenv .


FROM scratch

COPY --from=go /build/awsenv /awsenv

COPY --from=go /etc/ssl/certs /etc/ssl/certs

VOLUME /ssm

ENTRYPOINT ["/awsenv"]
