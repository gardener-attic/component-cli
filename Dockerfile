# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#### BUILDER ####
FROM golang:1.18.4 AS builder

WORKDIR /go/src/github.com/gardener/component-cli
COPY . .

ARG TARGETARCH
ARG EFFECTIVE_VERSION

RUN GOARCH=$TARGETARCH make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

#### BASE ####
FROM gcr.io/distroless/static-debian11:nonroot AS base

#### Component CLI ####
FROM base as cli

COPY --from=builder /go/bin/component-cli /component-cli

WORKDIR /

ENTRYPOINT ["/component-cli"]
