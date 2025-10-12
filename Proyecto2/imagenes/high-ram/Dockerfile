FROM alpine:3.20
RUN apk add --no-cache stress-ng
ENV RAM_MB=512 VM_WORKERS=1
# --vm-keep mantiene la memoria asignada; sin timeout
CMD ["sh","-c","exec stress-ng --vm ${VM_WORKERS} --vm-bytes ${RAM_MB}M --vm-keep --timeout 0"]
