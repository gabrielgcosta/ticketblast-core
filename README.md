# TicketBlast Core 🎟️⚡

TicketBlast Core is a high-concurrency, high-availability ticket-selling backend engine designed to handle massive traffic spikes without overselling inventory. This project is built following Clean Architecture principles, ensuring strict separation of concerns, high testability, and robustness.

> [!NOTE]
> This repository is currently in its initial bootstrapping phase. The documentation and codebase will evolve incrementally as new architectural layers are implemented.

---

## 🚀 Current Project State: Phase 1 (Foundation)

This initial commit establishes the basic foundation of the HTTP API Gateway using the **Gin Gonic** framework.

### Features in this Commit:
- **Project Structure:** Basic layout adhering to standard Go project layouts (`cmd/api`).
- **HTTP Server:** Instantiated Gin engine running on port `:8080`.
- **Health Check Probe:** A professional `/ping` endpoint to verify service liveness.

---

## 🛠️ Tech Stack

- **Language:** Go (Golang) 1.24+
- **Web Framework:** Gin Gonic
- **Dependency Management:** Go Modules

---

## 📋 API Endpoint Reference

### Health Check / Liveness Probe
In production-grade systems, a liveness/readiness probe is a standard pattern utilized by load balancers and orchestrators (e.g., Kubernetes, AWS ECS) to monitor application health and traffic routing.

* **URL:** `/ping`
* **Method:** `GET`
* **Response Content-Type:** `application/json`
* **Success Response:**
  ```json
  {
    "message": "pong"
  }
  ```

---

## 💻 How to Run Locally

### Prerequisites
- [Go 1.24+](https://go.dev/doc/install) installed on your machine.

### Execution Steps
1. Clone the repository and navigate to the root directory:
   ```bash
   cd ticketblast-core
   ```
2. Download dependencies:
   ```bash
   go mod download
   ```
3. Run the API Gateway:
   ```bash
   go run cmd/api/main.go
   ```
4. Test the health check endpoint:
   ```bash
   curl http://localhost:8080/ping
   ```
