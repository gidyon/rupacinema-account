FROM scratch

LABEL maintainer="gideonhacer@gmail.com"

WORKDIR /

COPY server.bin /
COPY certs /certs

ENTRYPOINT [ "/server.bin" ]