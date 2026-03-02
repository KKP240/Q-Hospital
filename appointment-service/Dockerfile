FROM golang:1.25-alpine

WORKDIR /app

# เพิ่ม build-base เผื่อ gorm/postgres ต้องการใช้
RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

CMD ["./main"]