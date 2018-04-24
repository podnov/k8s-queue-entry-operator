FROM golang:1.9-alpine

MAINTAINER Evan Zeimet <podnov@gmail.com>

RUN useradd queo
USER queo

COPY k8s-queue-entry-operator /home/queo/bin/
WORKDIR /home/queo

CMD /home/queo/bin/k8s-queue-entry-operator
