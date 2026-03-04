FROM golang:1.25-alpine

WORKDIR /app

# เพิ่ม build-base เผื่อ gorm/postgres ต้องการใช้
RUN apk add --no-cache build-base

COPY auth/ /auth/

COPY user-service/go.mod user-service/go.sum ./
RUN go mod edit -replace github.com/KKP240/Q-Hospital/auth=/auth
RUN go mod download

COPY user-service/ .
RUN go build -o main .

CMD ["./main"]