FROM golang:1.22 as build

ARG USERNAME=app
ARG USER_UID=42000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

WORKDIR /go/src/github.com/voidshard/totp
ADD go.mod go.sum ./
ADD *.go ./
ADD cmd cmd
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /totp ./cmd/totp && chown -R $USER_UID:$USER_GID /totp


FROM scratch

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /totp /totp

USER $USERNAME
ENTRYPOINT ["/totp"]
