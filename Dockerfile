FROM golang:1.21 as build

ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:3.19
COPY --from=build /app/ShadowTest /usr/bin/

ENV APP_USER=shadowtest
ENV APP_GROUP=shadowtestgroup
RUN addgroup -S $APP_GROUP && adduser -H -S $APP_USER -G $APP_GROUP -s /sbin/nologin
RUN chown -R $APP_USER:$APP_GROUP /usr/bin/ShadowTest
USER $APP_USER

EXPOSE 8080
HEALTHCHECK CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT /usr/bin/ShadowTest
