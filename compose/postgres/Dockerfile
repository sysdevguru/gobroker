FROM postgres:10.4-alpine

COPY server.crt /ssl/server.crt
COPY server.key /ssl/server.key

RUN chown postgres:postgres /ssl/server.* && chmod 600 /ssl/server.*
