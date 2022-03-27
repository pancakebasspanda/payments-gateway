# payments-gateway

**About**
The payments-gateway is a gRPC server implementation that allows 
a merchant to 
* Process a payment through your payment gateway.
* Retrieve details of a previously made payment.

The payment gateway repo also contains configuration for a Bank simulator that is part
of the payments lifecycle and used for validating and authorizing payments.

The Bank simulator was created using [MockServer](https://www.mock-server.com/)
Since a simulator would return specific outputs for a given input and it would be
an external service, MockServer was used due to its easy configuration and simplicity. 
See `config/initializerJson.json` for the configuration. The actual server is run in a docker container.

The payment-gateway service also interacts with a Postgres db that is defined in
`scripts/db/init.sql`. The db s responsible for storage of payment details. 

**How it Works**
In order to run the payments-gateway service, you first need to run the docker containers
running the mock-service and well as th acquiring bank simulator.
This is achieved by running the docker-compose.yaml file as follows

```shell
docker-compose up
```

This spins up containers for all the required services that the payment gateway will interact with.
Thereafter you can run the payments-gateway code either from your favourite IDE or via

```shell
$ go build cmd/payments-gateway/main.go

$ go run cmd/payments-gateway/main.go

{"level":"info","msg":"Starting payments-gateway gRPC server","port":9090,"time":"2022-03-27T22:52:48+01:00"}

```

## Generating the go files from the protos definition
```shell
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protos/payments.proto
```

## Running tests
```shell
go test ./...
```

## Packages

`/cmd`: main.go

`/aquiring-bank`: interface that has a client implementation for the acquiring bank simulation

`/server`: gRPC server implementation

`/protos`: protobuff definitions and generated go files for the gRPC server

`/storage`: Storage interface

`/storage/postgres`: Postgres db client implementation
