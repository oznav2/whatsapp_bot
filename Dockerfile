FROM golang:1.24.1

WORKDIR /app

COPY . .

RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y gcc libc6-dev python3 python3-pip ffmpeg wget
RUN pip3 install --break-system-packages yt-dlp curl-cffi
RUN go mod tidy
RUN go build -o main main.go

EXPOSE 8086

CMD ["./main"]
