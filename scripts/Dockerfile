FROM alpine:latest

WORKDIR /scripts

COPY *.sh .

CMD rm -rf /mnt/scripts/* && cp -r . /mnt/scripts
