#!/bin/bash

# vars
TARGET_PUSH=${TARGET_PUSH:-"docker://"}
TARGET_REGISTRY=${TARGET_REGISTRY:-"docker.io"}
TARGET_ORG=${TARGET_ORG:-"chrisruffalo"}
TARGET_REPO=${TARGET_REPO:-"gyip"}

# do a clean build (without pushing local artifacts)
source ./build.sh 0
echo "Pushing ${TARGET_REPO}/${TARGET_ORG}/${TARGET_REPO}:${BUILD_TAG}"

# login with podman
podman login ${TARGET_REGISTRY} --username "${DOCKERUSERNAME}" --password "${DOCKERPASSWORD}"

# push to remote
buildah push gyip/gyip:${BUILD_TAG} ${TARGET_PUSH}${TARGET_REGISTRY}/${TARGET_ORG}/${TARGET_REPO}:${BUILD_TAG}
buildah push gyip/gyip:${BUILD_TAG} ${TARGET_PUSH}${TARGET_REGISTRY}/${TARGET_ORG}/${TARGET_REPO}:${MAJOR_TAG}
buildah push gyip/gyip:${BUILD_TAG} ${TARGET_PUSH}${TARGET_REGISTRY}/${TARGET_ORG}/${TARGET_REPO}:latest