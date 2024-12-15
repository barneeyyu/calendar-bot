# Use the official AWS Lambda base image for Go
FROM public.ecr.aws/lambda/go:1

# Copy the Go application binary into the container
COPY ./bin/bootstrap ${LAMBDA_TASK_ROOT}/bootstrap

# Command for Lambda to run
ENTRYPOINT [ "/lambda-entrypoint.sh" ]
CMD [ "bootstrap" ]