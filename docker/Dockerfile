FROM golang
MAINTAINER Kohei Arao <koara.local.jp@gmail.com>

ENV DEBIAN_FRONTEND noninteractive

# update package
RUN { \
    echo "deb http://ftp.jp.debian.org/debian/ wheezy-backports main contrib non-free"; \
    echo "deb-src http://ftp.jp.debian.org/debian/ wheezy-backports main contrib non-free"; \
    } >> /etc/apt/sources.list.d/backports.list

RUN apt-get update -yq && \
    apt-get install -y --no-install-recommends nodejs nodejs-legacy coffeescript && \
    apt-get clean && \
    rm -rf /var/cache/apt/archives/* /var/lib/apt/lists/*

# irori
WORKDIR /root
RUN git config --global http.sslVerify false
RUN git clone --depth 1 --single-branch -b master https://github.com/maueki/irori.git
WORKDIR /root/irori
RUN make
EXPOSE 8000
