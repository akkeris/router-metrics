FROM golang:1.12
RUN apt-get update
RUN apt-get install -y tzdata git
RUN cp /usr/share/zoneinfo/America/Denver /etc/localtime
RUN mkdir -p /go/src/router-metrics
WORKDIR /go/src/router-metrics
ADD . .
RUN chmod +x ./build.sh
RUN ./build.sh
CMD ["/go/src/router-metrics/router-metrics"]




