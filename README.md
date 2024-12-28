# Vision One Container Security serverless application template

If you find issues or improvements, please contribute back to the template repo 🙏

_**Rewrite this README for your own application after initial setup**_

## Setup

1. Address all `TODO` comments.

2. Install [Node.js](https://nodejs.org/) and [Go](https://go.dev/).

3. Install and upgrade all packages to ensure your application is initialized with the latest package versions.  Note, this only need be done once.

       go get -t -u ./...
       go mod tidy

       npm update
       npx ncu -u

4. Commit changes

       git add .
       git commit

That's it, you are good to start coding!

For more on building AWS Lambdas with Go, see [AWS docs](https://docs.aws.amazon.com/lambda/latest/dg/lambda-golang.html)

## Development Commands

### Linting
Execute the linter to check code quality:

    make lint

Run linter with auto-fixing enabled:

    make lint-fix

### Testing
Run all tests:

    make test

Generate test coverage report:

    make coverage

### Building
Build the application:

    make docker-all      # Build Docker images
    make push-all        # Push images to ECR
    make build-and-push-all  # Build and push in one command

### Cleanup
Clean build artifacts:

    make clean           # Remove binaries
    make docker-clean    # Remove Docker images

## Deploy
### Deploy to AWS on local
Run the following command to deploy the application:

1. Login aws first, paste your aws credentials
```bash
aws configure
#and paste your keys
```

2.
    2.1Copy the environment template file:
    ```
    cp .env.example .env
    ```
    2.2 Edit the `.env` file and fill in your LINE Bot credentials:
    ```bash
    CHANNEL_SECRET={{your_channel_secret}}
    CHANNEL_TOKEN={{your_channel_token}}
    # These credentials can be found in the LINE Developers Console.
    ```

1. Deploy to AWS
```bash
sls deploy --stage {stageName} --verbose
```

1. When deploy successfully, copy the url of `line-events` API, and paste to the ***Line developers*** -> Webhook URL

### API Specification

This document outlines the usage and specifications of two APIs. These APIs are designed for handling events from the LINE platform and broadcasting messages to users subscribed to a specific official account.

### /line-events

This API acts as a webhook for the LINE Bot, receiving and processing various events from the LINE platform.

#### Request Method
`POST`
