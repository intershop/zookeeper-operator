#
# Copyright (c) 2018 Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#

ARG DOCKER_REGISTRY
# Runtime image (zookeeper:3.8.6) ships JDK 17; keep the build JDK aligned.
FROM  ${DOCKER_REGISTRY:+$DOCKER_REGISTRY/}eclipse-temurin:17-jdk
RUN mkdir /zu
COPY zu /zu
WORKDIR /zu
RUN ./gradlew --console=verbose --info shadowJar

FROM ${DOCKER_REGISTRY:+$DOCKER_REGISTRY/}zookeeper:3.8.6
COPY bin /usr/local/bin
RUN chmod +x /usr/local/bin/*
COPY --from=0 /zu/build/libs/zu.jar /opt/libs/

RUN apt-get -q update && \
        apt-get install -y dnsutils curl procps socat && \
        curl --fail --location --silent --show-error \
            --output /apache-zookeeper-3.8.6-bin/lib/logback-ecs-encoder-1.8.0.jar \
            https://repo1.maven.org/maven2/co/elastic/logging/logback-ecs-encoder/1.8.0/logback-ecs-encoder-1.8.0.jar && \
        curl --fail --location --silent --show-error \
            --output /apache-zookeeper-3.8.6-bin/lib/ecs-logging-core-1.8.0.jar \
            https://repo1.maven.org/maven2/co/elastic/logging/ecs-logging-core/1.8.0/ecs-logging-core-1.8.0.jar && \
        rm -rf /var/lib/apt/lists/*
