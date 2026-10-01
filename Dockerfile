FROM golang:1.23-alpine

WORKDIR /app 
COPY go.mod go.sum ./

RUN go mod download

copy . .

RUN go build -o /kademlia ./cmd

EXPOSE 8080

CMD ["sh", "-c", "IP=$(hostname -i); echo \"Node address: ${IP}\"; echo \"port:8080\"; /kademlia start --ip \"$IP\" --port 8080 --bootstrap bootstrap:8080"]


# Add the commands needed to put your compiled go binary in the container and
# run it when the container starts.
#
# See https://docs.docker.com/engine/reference/builder/ for a reference of all
# the commands you can use in this file.
#
# In order to use this file together with the docker-compose.yml file in the
# same directory, you need to ensure the image you build gets the name
# "kadlab", which you do by using the following command:
#
# $ docker build . -t kadlab