FROM alpine

RUN apk update && apk add ca-certificates

COPY kube_scheduler_extender /
RUN chmod +x /kube_scheduler_extender
WORKDIR /
ENTRYPOINT ["./kube_scheduler_extender"]