Mkwanja Payment Service
========================
### Overview

The Mkwanja Payment Service is a microservice designed to handle all payment processing for the Mkwanja app. It integrates multiple payment gateways, including M-PESA, Flutterwave, and Mookh, to provide a seamless payment experience.

### Technology Stack

* **Programming Language:** Go
* **Web Framework:** [Fiber](https://gofiber.io/)
* **Development Tools:**
  1. [Fresh](https://github.com/gravityblast/fresh)
  2. [GORM](https://gorm.io/)
  3. [godotenv](https://github.com/joho/godotenv)

### Getting Started

#### Prerequisites

* Go (version 1.23.0 or higher) installed on your system
* Fresh installed on your system (`go install github.com/gravityblast/fresh@latest`)
* GORM installed on your system (`go get -u gorm.io/gorm`)
* A code editor or IDE of your choice

#### Running the Project

1. Clone the repository: `git clone https://github.com/dexterbrian/mkwanja-payment-service.git`
2. Navigate to the project directory: `cd mkwanja-payment-service`
3. Install libraries by running: `go mod tidy`
4. Run the project using Fresh: `fresh`
5. The project will start in development mode, watching for file changes and automatically reloading the server.

### Configuration

The project uses JSON configuration files stored in the `config/` directory in combination with environment variables to configure various aspects of the application. Environment variables are found in the root of the project's directory in the `.env` file.

### Database Setup

The project uses MySQL as its database. Configuration for the database connection can be found in the `config/database.go` file. To set up the database:

1. Ensure PostgreSQL is installed and running on your system.
2. Update the `.env` file with your database credentials.
3. Run migrations using the provided migration script: `go run scripts/migrate.go`
4. Seed the database: `go run scripts/seed.go`

### Testing

Tests are located in the `tests/` directory and are organized by payment gateway. To run the tests:

1. Navigate to the project directory: `cd mkwanja-payment-service`
2. Run the tests using the Go test tool: `go test ./...`

### Deployment

Deployment details are specific to the environment. For a typical deployment:

1. Build the project: `go build -o mkwanja-payment-service`
2. Deploy the binary to your server.
3. Run the binary: `./mkwanja-payment-service`

### Swagger API Documentation (coming soon)

API documentation is generated using Swagger and can be found at `http://localhost:3000/swagger/index.html` when the project is running in development mode.

### Contributing

Contributions are welcome! Please submit a pull request with a clear description of the changes made.

### Author

This project was authored by Dexter Brian Waweru and is maintained by the Mkwanja team.

### License

This project is licensed under the MIT License.

### Acknowledgments

* The Mkwanja team for their hard work and dedication.
* The Go and Fiber communities for their support and resources.
