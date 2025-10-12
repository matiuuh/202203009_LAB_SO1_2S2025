FROM alpine:3.20
RUN apk add --no-cache coreutils
ENV CPU_WORKERS=1
# Lanza N procesos "yes" para forzar CPU; espera a todos
CMD ["sh","-c","i=0; while [ $i -lt ${CPU_WORKERS} ]; do yes > /dev/null & i=$((i+1)); done; wait"]
