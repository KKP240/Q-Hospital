# 🏥 Hospital Queue Microservices

ระบบจัดการคิวโรงพยาบาลแบบ **Microservices Architecture**
สำหรับรายวิชา **Microservice Design and Development**

------------------------------------------------------------------------

## 📌 Architecture Overview

    ┌─────────────────┐      ┌──────────────────┐      ┌─────────────────────┐
    │ Appointment     │────▶│ Queue-Patient     │────▶│ Payment-Pill        │
    │ Service :8081   │      │ Service :8082     │      │ Service :8083       │
    └─────────────────┘      └──────────────────┘      └─────────────────────┘
                │                         │                         │
                └─────────────────────────┴─────────────────────────┘
                                   │
                             ┌─────────────┐
                             │ RabbitMQ    │
                             │ :5672       │
                             └─────────────┘

------------------------------------------------------------------------

## 📋 สารบัญ

-   เทคโนโลยีที่ใช้\
-   การติดตั้ง\
-   วิธีใช้งาน\
-   API Documentation\
-   Architecture

------------------------------------------------------------------------

## 🛠 เทคโนโลยีที่ใช้

  Component       | Technology      |          Port |
  ----------------| -------------------------| --------------
  API Gateway    |  Gin (Go)      |            8081-8083|
  Message Broker |  RabbitMQ       |           5672 / 15672
  Database       |  PostgreSQL       |         5432
  Container   |     Docker & Docker Compose   |\-

------------------------------------------------------------------------

## 🚀 การติดตั้ง

### ขั้นตอนที่ 1: Clone Repository

``` bash
git clone https://github.com/KKP240/Q-Hospital.git
cd Q-Hospital
```
### ขั้นตอนที่ 2: ติดตั้ง dependecies

วิธีแรก:

``` bash
go mod tidy
```

หรือรัน:

``` bash
go mod download
```

### ขั้นตอนที่ 3: รัน Services

รันทั้งหมดพร้อมกัน:

``` bash
docker-compose up --build
```

หรือรันแบบ background:

``` bash
docker-compose up -d --build
```

### ขั้นตอนที่ 3: ตรวจสอบว่าพร้อมใช้งาน

รอประมาณ 10-15 วินาที แล้วตรวจสอบ:

  Service              |   URL                       |     สถานะ |
  -----------------------| ------------------------------| -------------
  RabbitMQ Management   | http://localhost:15672       |  admin/admin
  Appointment Service    | http://localhost:8081/health  | OK
  Queue-Patient Service  | http://localhost:8082/health  | OK
  Payment-Pill Service   | http://localhost:8083/health  | OK

------------------------------------------------------------------------

## 📡 วิธีใช้งาน (Flow หลักของระบบ)

    [สร้างนัดหมาย] → [ยืนยันนัด] → [สร้างคิวอัตโนมัติ] → [เรียกคิว] → [ตรวจเสร็จ]
            ↓
    [จบการรักษา] ← [จ่ายยา] ← [สร้างใบสั่งยาอัตโนมัติ] ← [จ่ายเงิน] ← [สร้างใบเสร็จอัตโนมัติ]

------------------------------------------------------------------------

## 🧪 ทดสอบด้วย cURL

### 1️⃣ สร้างนัดหมาย

``` bash
curl -X POST http://localhost:8081/appointments -H "Content-Type: application/json" -d '{ "patient": "สมชาย ใจดี", "doctor": "นพ.สมหญิง รักษาดี", "date": "2025-03-15" }'
```

Response:

``` json
{
  "id": 1,
  "patient": "สมชาย ใจดี",
  "doctor": "นพ.สมหญิง รักษาดี",
  "date": "2025-03-15",
  "status": "pending"
}
```

------------------------------------------------------------------------

### 2️⃣ ยืนยันนัดหมาย (สร้างคิวอัตโนมัติ)

``` bash
curl -X PUT http://localhost:8081/appointments/1/confirm
```

------------------------------------------------------------------------

### 3️⃣ ตรวจสอบคิวที่สร้าง

``` bash
curl http://localhost:8082/queues
```

------------------------------------------------------------------------

### 4️⃣ เรียกผู้ป่วยเข้าห้องตรวจ

``` bash
curl -X PUT http://localhost:8082/queues/1/call
```

------------------------------------------------------------------------

### 5️⃣ ตรวจเสร็จ (สร้างใบเสร็จอัตโนมัติ)

``` bash
curl -X PUT http://localhost:8082/queues/1/done
```

------------------------------------------------------------------------

### 6️⃣ ตรวจสอบใบเสร็จ

``` bash
curl http://localhost:8083/payments/1
```

------------------------------------------------------------------------

### 7️⃣ ชำระเงิน (สร้างใบสั่งยาอัตโนมัติ)

``` bash
curl -X PUT http://localhost:8083/payments/1/pay
```

------------------------------------------------------------------------

### 8️⃣ จ่ายยาเสร็จสิ้น

``` bash
curl -X PUT http://localhost:8083/prescriptions/1/dispense
```

------------------------------------------------------------------------

## 📚 API Documentation

### 🔹 Appointment Service (:8081)

  Method |  Endpoint            |        Description
  --------| --------------------------- |--------------------------
  POST  |   /appointments             |  สร้างนัดหมายใหม่
  GET   |   /appointments              | ดูรายการนัดหมายทั้งหมด
  PUT   |   /appointments/:id/confirm  | ยืนยันนัดหมาย → สร้างคิว
  PUT   |   /appointments/:id/cancel  |  ยกเลิกนัดหมาย

------------------------------------------------------------------------

### 🔹 Queue-Patient Service (:8082)

  Method  | Endpoint         |  Description
  --------| ------------------ |--------------------------
  GET     | /queues           | ดูคิวปัจจุบันทั้งหมด
  PUT    |  /queues/:id/call  | เรียกผู้ป่วยเข้าห้องตรวจ
  PUT    |  /queues/:id/done  | ตรวจเสร็จ → สร้างใบเสร็จ

------------------------------------------------------------------------

### 🔹 Payment-Pill Service (:8083)

  Method  | Endpoint                   |  Description
  --------| -----------------------------| --------------------------
  GET    |  /payments/:queue_id         |  ดูใบเสร็จของคิว
  PUT    |  /payments/:id/pay            | ชำระเงิน → สร้างใบสั่งยา
  GET    |  /prescriptions/:queue_id     | ดูใบสั่งยา
  PUT    |  /prescriptions/:id/dispense  | จ่ายยาเสร็จสิ้น

------------------------------------------------------------------------

## 🏗 Architecture

    Client (Postman/cURL)
            │
            ▼
    ┌──────────────┐   ┌────────────────┐   ┌─────────────────┐
    │ Appointment  │──▶│ Queue-Patient  │──▶│ Payment-Pill    │
    │ Service      │   │ Service        │   │ Service         │
    └──────────────┘   └────────────────┘   └─────────────────┘
            │                  │                    │
            └──────────────▶ RabbitMQ ◀─────────────┘
                               │
                          PostgreSQL

### Communication Pattern

  ---------------------------------------------------------------------------------
  From        |      To          |    Event                |   Action
  -----------------| ---------------| -----------------------| -----------------------
  Appointment     |  Queue-Patient  | appointment.confirmed |  สร้างคิวอัตโนมัติ
  Queue-Patient   |  Payment-Pill  |  queue.done      |       สร้างใบเสร็จอัตโนมัติ
  Payment-Pill    |  \-            |  Internal         |       จ่ายเงิน →สร้างใบสั่งยา
  ---------------------------------------------------------------------------------

------------------------------------------------------------------------

## 🔧 การพัฒนาต่อ

### โครงสร้างโฟลเดอร์

    hospital-3services/
    ├── docker-compose.yml
    ├── appointment-service/
    │   ├── main.go
    │   └── Dockerfile
    ├── queue-patient-service/
    │   ├── main.go
    │   └── Dockerfile
    └── payment-pill-service/
        ├── main.go
        └── Dockerfile

### Rebuild

``` bash
docker-compose up --build
```

เฉพาะ service:

``` bash
docker-compose up --build appointment-service
```

### Logs

``` bash
docker-compose logs -f
docker-compose logs -f appointment-service
```

### Reset Database

``` bash
docker-compose down -v
docker-compose up --build
```

------------------------------------------------------------------------

## 🐛 Troubleshooting

  ปัญหา             |    แก้ไข
  --------------------- |---------------------------------------------
  connection refused  |  รอ 10 วินาทีให้ Database พร้อม
  queue not created   |  เช็ค RabbitMQ ที่ http://localhost:15672
  port already in use  | ปิดโปรแกรมที่ใช้ port 8081-8083, 5432, 5672

------------------------------------------------------------------------

## 👥 ผู้จัดทำ

รายวิชา 06016428\
Microservice Design and Development\
ปีการศึกษา 2/2568

------------------------------------------------------------------------

## 📝 License

MIT License - สามารถนำไปพัฒนาต่อได้ฟรี
