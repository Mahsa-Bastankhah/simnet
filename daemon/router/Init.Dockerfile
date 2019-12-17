FROM alpine

LABEL maintainer="DEDIS <dedis@epfl.ch>"

EXPOSE 1194/udp

VOLUME ["/etc/openvpn"]

RUN apk add --no-cache easy-rsa

WORKDIR /app
COPY ./daemon/router/init.sh .
COPY ./daemon/router/dh.pem .

ENTRYPOINT [ "./init.sh" ]
