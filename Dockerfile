# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#### BUILDER ####
FROM eu.gcr.io/gardener-project/3rd/golang:1.15.5 AS builder

WORKDIR /go/src/github.com/gardener/component-cli
COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

#### BASE ####
FROM eu.gcr.io/gardenlinux/gardenlinux:184.0 AS base

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get --yes -o Dpkg::Options::="--force-confnew" install ca-certificates \
    && rm -rf /var/lib/apt /var/cache/apt

#### Component CLI ####
FROM base as cli

COPY --from=builder /go/bin/component-cli /component-cli

WORKDIR /

ENTRYPOINT ["/component-cli"]
