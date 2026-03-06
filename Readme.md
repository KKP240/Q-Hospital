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

# 🏥 Clinic API Testing Guide (Postman)

คู่มือสำหรับการทดสอบระบบ API ของคลินิกผ่าน Postman โดยแบ่งตาม Service ต่างๆ และมีการจัดการสิทธิ์ผู้ใช้งาน (Role-based Access Control)

---

## 🔑 การตั้งค่า Token (Authentication)
ระบบนี้ใช้ **JWT Token** ในการยืนยันตัวตน หลังจากทำการ Login สำเร็จ ให้นำ Token ที่ได้ไปตั้งค่าใน Postman ดังนี้:
1. ไปที่แท็บ **Headers** ของ Request
2. เพิ่ม Key: `Authorization`
3. เพิ่ม Value: `Bearer <ใส่_Token_ที่นี่>`
*(หรือใช้แท็บ **Authorization** -> เลือก Type เป็น **Bearer Token**)*

---

## 👤 1. Authentication Service (Port `8084`)

### 1.1 ลงทะเบียนผู้ใช้งาน (Register)
* **Method:** `POST`
* **URL:** `http://localhost:8084/register`
* **Body (JSON):**

**สำหรับผู้ป่วย (Patient):**
```json
{
    "name": "patient1",
    "email": "patient1@gmail.com",
    "password": "1234",
    "role": "patient",
    "address": "abcs",
    "phone_number": "0649515415"
}
```
**สำหรับแพทย์ (Doctor):**
```json
{
    "name": "doctor1",
    "email": "doctor1@gmail.com",
    "password": "1234",
    "role": "doctor",
    "address": "abcs",
    "phone_number": "0323243423"
}
```

### 1.2 เข้าสู่ระบบ (Login)
* **Method:** `POST`
* **URL:** `http://localhost:8084/login`
* **Body (JSON):**

```json
{
    "email": "patient1@gmail.com",
    "password": "1234"
}
```
(ใช้ email/password ของบัญชีที่ต้องการ Login เพื่อรับ Token นำไปใช้ในขั้นตอนต่อไป)

## 📅 2. Appointment Service (Port 8081)

### 2.1 สร้างนัดหมาย (Create Appointment)
* **Method:** `POST`
* **URL:** `http://localhost:8081/appointments`
* **Access:** ⚠️ เฉพาะ Doctor เท่านั้น (ต้องใส่ Token ของ Doctor)
* **Body (JSON):**

```json
{
    "patient_id": "04db5cf1-66df-4b85-993b-63db71ef8c98",
    "doctor_id": "b07ce4d2-16cc-4fb4-94e3-fa00288c5c4e",
    "date": "2025-03-15"
}
```

### 2.2 ดูนัดหมายทั้งหมด (Get Appointments)
* **Method:** `GET`
* **URL:** `http://localhost:8081/appointments`
* **Access:** ✅ ดูได้ทั้ง Doctor และ Patient (ต้องแนบ Token)

### 2.3 ยืนยันนัดหมาย (Confirm Appointment) - สร้างคิวอัตโนมัติ
* **Method:** `PUT`
* **URL:** `http://localhost:8081/appointments/:id/confirm` (แทนที่ :id ด้วย ID ของนัดหมาย)
* **Access:** ⚠️ เฉพาะ Patient เท่านั้น (Doctor ไม่สามารถแก้ไขได้)

### 2.4 ยกเลิกนัดหมาย (Cancel Appointment)
* **Method:** `PUT`
* **URL:** `http://localhost:8081/appointments/:id/cancel` (แทนที่ :id ด้วย ID ของนัดหมาย)
* **Access:** ⚠️ เฉพาะ Patient เท่านั้น (Doctor ไม่สามารถแก้ไขได้)

## 🚶‍♂️ 3. Queue Service (Port 8082)

### 3.1 ตรวจสอบคิวที่สร้าง
* **Method:** `GET`
* **URL:** `http://localhost:8082/queues`

### 3.2 เรียกผู้ป่วยเข้าห้องตรวจ
* **Method:** `PUT`
* **URL:** `http://localhost:8082/queues/1/call`

### 3.3 ตรวจเสร็จสิ้น (สร้างใบเสร็จอัตโนมัติ)
* **Method:** `PUT`
* **URL:** `http://localhost:8082/queues/1/done`

## 💊 4. Payment & Pill Service (Port 8083)

### 4.1 ตรวจสอบข้อมูลผู้ใช้ปัจจุบัน
* **Method:** `GET`
* **URL:** `http://localhost:8083/my`
* **Auth:** `Doctor, Patient`

### 4.2 ดู Payments ทั้งหมด (เฉพาะ Doctor)
* **Method:** `GET`
* **URL:** `http://localhost:8083/payments`
* **Auth:** `Doctor only`

### 4.3 ดู Prescriptions ทั้งหมด (เฉพาะ Doctor)
* **Method:** `GET`
* **URL:** `http://localhost:8083/prescriptions`
* **Auth:** `Doctor only`

### 4.4 ดู Payments ของตัวเอง (เฉพาะ Patient)
* **Method:** `GET`
* **URL:** `http://localhost:8083/my/payments`
* **Auth:** `Patient only`

### 4.5 ดู Prescriptions ของตัวเอง (เฉพาะ Patient)
* **Method:** `GET`
* **URL:** `http://localhost:8083/my/prescriptions`
* **Auth:** `Patient only`

### 4.6 ดู Payment รายการเดียว (Doctor ดูได้ทุกอัน / Patient ดูได้แค่ของตัวเอง)
* **Method:** `GET`
* **URL:** `http://localhost:8083/payments/{queue_id}`
* **Auth:** `Doctor, Patient`

### 4.7 ดู Prescription รายการเดียว (Doctor ดูได้ทุกอัน / Patient ดูได้แค่ของตัวเอง)
* **Method:** `GET`
* **URL:** `http://localhost:8083/prescriptions/{queue_id}`
* **Auth:** `Doctor, Patient`

### 4.8 ชำระเงิน (สร้างใบสั่งยาอัตโนมัติ / จ่ายได้แค่ของตัวเอง)
* **Method:** `PUT`
* **URL:** `http://localhost:8083/payments/{id}/pay`
* **Auth:** `Patient only`

### 4.9 จ่ายยาเสร็จสิ้น (เฉพาะ Doctor)
* **Method:** `PUT`
* **URL:** `http://localhost:8083/prescriptions/{id}/dispense`
* **Auth:** `Doctor only`

------------------------------------------------------------------------

## 📚 API Documentation

### 🔹 User Service (:8084)

Method | Endpoint | Description
--------|----------------|--------------------------
POST | /register | สมัครสมาชิกผู้ใช้ใหม่
POST | /login | เข้าสู่ระบบและรับ Token
GET | /users | ดูรายการผู้ใช้ทั้งหมด
GET | /users/:id | ดูข้อมูลผู้ใช้ตาม ID
GET | /patients/:id | ดูข้อมูลผู้ป่วยตาม ID
GET | /doctors/:id | ดูข้อมูลแพทย์ตาม ID

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

Method | Endpoint | Description
-------|----------|------------
GET | /me | ดูข้อมูลผู้ใช้ปัจจุบัน
GET | /payments | ดู Payments ทั้งหมด (Doctor only)
GET | /prescriptions | ดู Prescriptions ทั้งหมด (Doctor only)
GET | /me/payments | ดู Payments ของตัวเอง (Patient only)
GET | /me/prescriptions | ดู Prescriptions ของตัวเอง (Patient only)
GET | /payments/:queue_id | ดู Payment รายการเดียว (Doctor ดูได้ทุกอัน / Patient ดูได้แค่ของตัวเอง)
GET | /prescriptions/:queue_id | ดู Prescription รายการเดียว (Doctor ดูได้ทุกอัน / Patient ดูได้แค่ของตัวเอง)
PUT | /payments/:id/pay | ชำระเงิน → สร้างใบสั่งยาอัตโนมัติ (Patient only)
PUT | /prescriptions/:id/dispense | จ่ายยาเสร็จสิ้น (Doctor only)

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

    QUEUE-HOSPITAL/
    ├── docker-compose.yml
    ├── appointment-service/
    │   ├── main.go
    │   ├── go.mod
    │   └── go.sum
    ├── auth/
    │   ├── auth.go
    │   ├── go.mod
    │   ├── go.sum
    │   └── middleware/
    │       └── gin.go
    ├── init-db/
    │   └── init.sql
    ├── payment-pill-service/
    │   ├── main.go
    │   ├── Dockerfile
    │   ├── go.mod
    │   └── go.sum
    ├── queue-patient-service/
    │   ├── main.go
    │   ├── Dockerfile
    │   ├── go.mod
    │   └── go.sum
    └── user-service/
        ├── main.go
        ├── Dockerfile
        ├── go.mod
        └── go.sum

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
