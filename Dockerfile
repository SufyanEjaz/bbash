#
# Copyright (c) 2021-present Sonatype, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

FROM node:16.13.2-alpine3.15 as yarn-build
LABEL stage=builder

RUN apk add --update build-base

WORKDIR /src

COPY . .

RUN make yarn

FROM golang:1.16.2-alpine AS build
LABEL stage=builder

RUN apk add --update build-base ca-certificates git

ENV USER=bbashuser
ENV UID=10001 

WORKDIR /src

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

COPY . .

# Ensures that the build from yarn is used, not an existing build on the local device
COPY --from=yarn-build /src/build /src/build

RUN make go-alpine-build

FROM scratch AS bin
LABEL application=bug-bash

EXPOSE 7777

WORKDIR /

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /src/build /build
COPY --from=build /src/bbash /
COPY *.env /
COPY *.env.dd /
ADD internal/db/migrations /internal/db/migrations

USER bbashuser:bbashuser

ENTRYPOINT [ "./bbash" ]
