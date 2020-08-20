FROM envoyproxy/envoy-dev:a6f5d4b0d310d64a800a7592fa5182975efb0a0a

RUN sed -i s:/archive.ubuntu.com:/mirrors.tuna.tsinghua.edu.cn/ubuntu:g /etc/apt/sources.list
RUN cat /etc/apt/sources.list
RUN apt-get clean
RUN apt-get update
COPY envoy/envoy.yaml /etc/envoy/envoy.yaml

RUN mkdir /scr
COPY ctr /scr/ctr
COPY sidecar.sh /scr/sidecar.sh

CMD bash /scr/sidecar.sh
