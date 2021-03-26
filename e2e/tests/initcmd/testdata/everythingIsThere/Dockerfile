FROM golang:1.12

# Create project directory (workdir)
RUN mkdir -p /go/src/app
WORKDIR /go/src/app

# Add source code files to WORKDIR
ADD . .

# Install dependencies
RUN go get ./... 

# Build application (test if everything works)
RUN go build -o /tmp/my-app && chmod +x /tmp/my-app

# Install CompileDaemon for hot reloading
RUN go get github.com/githubnemo/CompileDaemon

# Start CompileDaemon for hot reloading (will watch for file changes and then rebuild & restart the application)
ENTRYPOINT CompileDaemon -log-prefix=false -color=true -build="go build -o /tmp/my-app" -command="/tmp/my-app"
