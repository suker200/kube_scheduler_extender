FROM alpine

COPY kube_scheduler_extender /
RUN chmod +x /kube_scheduler_extender
WORKDIR /
ENTRYPOINT ["./kube_scheduler_extender"]